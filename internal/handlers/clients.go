package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/db"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ClientsHandler handles all customer relationship routes.
type ClientsHandler struct {
	app *app.App
}

// NewClientsHandler creates a new ClientsHandler.
func NewClientsHandler(a *app.App) *ClientsHandler {
	return &ClientsHandler{app: a}
}

// ClientViewModel holds view details for a single client row.
type ClientViewModel struct {
	ID          string
	CompanyName string
	ContactName string
	Email       string
	Phone       string
	Category    string
	IsActive    bool
	TotalBilled int64
	BalanceDue  int64
}

// List queries and displays all active clients.
func (h *ClientsHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	query := `
		SELECT 
			c.id, COALESCE(c.company_name, ''), c.contact_name, COALESCE(c.email, ''), 
			COALESCE(c.phone, ''), COALESCE(c.category, ''), c.is_active,
			COALESCE((SELECT SUM(total) FROM invoices WHERE client_id = c.id), 0) as total_billed,
			COALESCE((SELECT SUM(balance_due) FROM invoices WHERE client_id = c.id AND status != 'cancelled'), 0) as balance_due
		FROM clients c
		WHERE c.tenant_id = ? AND c.is_active = 1
		ORDER BY c.contact_name ASC
	`

	rows, err := h.app.DB.QueryContext(r.Context(), query, tenant.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []ClientViewModel
	for rows.Next() {
		var c ClientViewModel
		var activeInt int
		err := rows.Scan(
			&c.ID, &c.CompanyName, &c.ContactName, &c.Email,
			&c.Phone, &c.Category, &activeInt, &c.TotalBilled, &c.BalanceDue,
		)
		if err == nil {
			c.IsActive = activeInt == 1
			clients = append(clients, c)
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "clients/list.html", clients, "Clients CRM", "clients")
}

// ShowNewForm renders the Client creation form.
func (h *ClientsHandler) ShowNewForm(w http.ResponseWriter, r *http.Request) {
	Render(w, r, h.app.TemplatesFS, "base", "clients/form.html", map[string]interface{}{}, "New Client", "clients")
}

// Create inserts a new Client profile into SQLite.
func (h *ClientsHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	contactName := r.FormValue("contact_name")
	if contactName == "" {
		SetFlash(w, "flash_error", "Contact name is required.")
		http.Redirect(w, r, "/clients/new", http.StatusSeeOther)
		return
	}

	clientID := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO clients (
			id, tenant_id, company_name, contact_name, email, phone, whatsapp, website, tax_id,
			billing_street, billing_city, billing_state, billing_zip, billing_country,
			shipping_street, shipping_city, shipping_state, shipping_zip, shipping_country,
			preferred_currency, payment_terms_days, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		clientID, tenant.ID,
		r.FormValue("company_name"),
		contactName,
		r.FormValue("email"),
		r.FormValue("phone"),
		r.FormValue("whatsapp"),
		r.FormValue("website"),
		r.FormValue("tax_id"),
		r.FormValue("billing_street"),
		r.FormValue("billing_city"),
		r.FormValue("billing_state"),
		r.FormValue("billing_zip"),
		r.FormValue("billing_country"),
		r.FormValue("shipping_street"),
		r.FormValue("shipping_city"),
		r.FormValue("shipping_state"),
		r.FormValue("shipping_zip"),
		r.FormValue("shipping_country"),
		r.FormValue("preferred_currency"),
		r.FormValue("payment_terms_days"),
		now, now,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/clients/new", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Client added successfully.")
	http.Redirect(w, r, "/clients/"+clientID, http.StatusSeeOther)
}

// ClientLedgerData aggregates performance stats, transactions, and documents for a client.
type ClientLedgerData struct {
	Client               ClientViewModel
	PreferredCurrency    string
	Phone                string
	Whatsapp             string
	Website              string
	TaxID                string
	BillingAddress       string
	ShippingAddress      string
	InvoicesCount        int
	InvoicesPaidCount    int
	BilledTotal          int64
	PaidTotal            int64
	OutstandingTotal     int64
	QuotesCount          int
	QuotesAcceptedCount  int
	QuoteConversionRate  float64
	Invoices             []ClientInvoiceRow
	Quotes               []ClientQuoteRow
	Payments             []ClientPaymentRow
}

type ClientInvoiceRow struct {
	ID            string
	InvoiceNumber string
	IssueDate     string
	DueDate       string
	Total         int64
	BalanceDue    int64
	Status        string
}

type ClientQuoteRow struct {
	ID          string
	QuoteNumber string
	IssueDate   string
	Total       int64
	Status      string
}

type ClientPaymentRow struct {
	ID            string
	Amount        int64
	Currency      string
	PaidAt        string
	MethodName    string
	Ref           string
}

// Show renders the detailed customer profile and ledger summaries.
func (h *ClientsHandler) Show(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientID := chi.URLParam(r, "id")
	ctx := r.Context()

	var data ClientLedgerData
	var activeInt int
	var companyNameOpt, emailOpt, phoneOpt, whatsappOpt, websiteOpt, taxIdOpt sql.NullString
	var bStreet, bCity, bState, bZip, bCountry sql.NullString
	var sStreet, sCity, sState, sZip, sCountry sql.NullString
	var categoryOpt sql.NullString
	var paymentTerms int

	query := `
		SELECT 
			id, COALESCE(company_name, ''), contact_name, email, phone, whatsapp, website, tax_id,
			billing_street, billing_city, billing_state, billing_zip, billing_country,
			shipping_street, shipping_city, shipping_state, shipping_zip, shipping_country,
			preferred_currency, payment_terms_days, category, is_active
		FROM clients
		WHERE tenant_id = ? AND id = ?
	`

	err := h.app.DB.QueryRowContext(ctx, query, tenant.ID, clientID).Scan(
		&data.Client.ID, &companyNameOpt, &data.Client.ContactName, &emailOpt, &phoneOpt, &whatsappOpt, &websiteOpt, &taxIdOpt,
		&bStreet, &bCity, &bState, &bZip, &bCountry,
		&sStreet, &sCity, &sState, &sZip, &sCountry,
		&data.PreferredCurrency, &paymentTerms, &categoryOpt, &activeInt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SetFlash(w, "flash_error", "Client not found.")
			http.Redirect(w, r, "/clients", http.StatusSeeOther)
			return
		}
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}

	data.Client.CompanyName = companyNameOpt.String
	data.Client.Email = emailOpt.String
	data.Client.Phone = phoneOpt.String
	data.Whatsapp = whatsappOpt.String
	data.Website = websiteOpt.String
	data.TaxID = taxIdOpt.String
	data.Client.Category = categoryOpt.String
	data.Client.IsActive = activeInt == 1

	// Structure addresses
	addr := func(st, ct, sa, zp, co sql.NullString) string {
		var parts []string
		if st.String != "" { parts = append(parts, st.String) }
		if ct.String != "" { parts = append(parts, ct.String) }
		if sa.String != "" { parts = append(parts, sa.String) }
		if zp.String != "" { parts = append(parts, zp.String) }
		if co.String != "" { parts = append(parts, co.String) }
		if len(parts) == 0 {
			return "No address on file"
		}
		return fmt.Sprintf("%s", strings.Join(parts, ", "))
	}

	data.BillingAddress = addr(bStreet, bCity, bState, bZip, bCountry)
	data.ShippingAddress = addr(sStreet, sCity, sState, sZip, sCountry)

	// Invoices count, Billed sum, Outstanding sum
	_ = h.app.DB.QueryRowContext(ctx, `
		SELECT COUNT(id), COALESCE(SUM(total), 0)
		FROM invoices
		WHERE tenant_id = ? AND client_id = ? AND status != 'cancelled'
	`, tenant.ID, clientID).Scan(&data.InvoicesCount, &data.BilledTotal)

	_ = h.app.DB.QueryRowContext(ctx, `
		SELECT COUNT(id) FROM invoices WHERE tenant_id = ? AND client_id = ? AND status = 'paid'
	`, tenant.ID, clientID).Scan(&data.InvoicesPaidCount)

	_ = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance_due), 0) FROM invoices WHERE tenant_id = ? AND client_id = ? AND status IN ('pending', 'sent', 'partially_paid', 'overdue')
	`, tenant.ID, clientID).Scan(&data.OutstandingTotal)

	data.PaidTotal = data.BilledTotal - data.OutstandingTotal
	if data.PaidTotal < 0 {
		data.PaidTotal = 0
	}

	// Quotes count, accepted conversion rate
	_ = h.app.DB.QueryRowContext(ctx, `
		SELECT COUNT(id) FROM quotes WHERE tenant_id = ? AND client_id = ?
	`, tenant.ID, clientID).Scan(&data.QuotesCount)

	_ = h.app.DB.QueryRowContext(ctx, `
		SELECT COUNT(id) FROM quotes WHERE tenant_id = ? AND client_id = ? AND status IN ('accepted', 'converted')
	`, tenant.ID, clientID).Scan(&data.QuotesAcceptedCount)

	if data.QuotesCount > 0 {
		data.QuoteConversionRate = (float64(data.QuotesAcceptedCount) / float64(data.QuotesCount)) * 100.0
	}

	// Load Invoices
	invRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, invoice_number, issue_date, due_date, total, balance_due, status
		FROM invoices
		WHERE tenant_id = ? AND client_id = ?
		ORDER BY issue_date DESC
	`, tenant.ID, clientID)
	if err == nil {
		defer invRows.Close()
		for invRows.Next() {
			var ir ClientInvoiceRow
			if err := invRows.Scan(&ir.ID, &ir.InvoiceNumber, &ir.IssueDate, &ir.DueDate, &ir.Total, &ir.BalanceDue, &ir.Status); err == nil {
				data.Invoices = append(data.Invoices, ir)
			}
		}
	}

	// Load Quotes
	qRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, quote_number, issue_date, total, status
		FROM quotes
		WHERE tenant_id = ? AND client_id = ?
		ORDER BY issue_date DESC
	`, tenant.ID, clientID)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			var qr ClientQuoteRow
			if err := qRows.Scan(&qr.ID, &qr.QuoteNumber, &qr.IssueDate, &qr.Total, &qr.Status); err == nil {
				data.Quotes = append(data.Quotes, qr)
			}
		}
	}

	// Load Payments
	pRows, err := h.app.DB.QueryContext(ctx, `
		SELECT p.id, p.amount, p.currency, p.paid_at, COALESCE(pm.name, 'Direct Method'), COALESCE(p.transaction_ref, '')
		FROM payments p
		LEFT JOIN payment_methods pm ON p.payment_method_id = pm.id
		WHERE p.tenant_id = ? AND p.client_id = ?
		ORDER BY p.paid_at DESC
	`, tenant.ID, clientID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pr ClientPaymentRow
			if err := pRows.Scan(&pr.ID, &pr.Amount, &pr.Currency, &pr.PaidAt, &pr.MethodName, &pr.Ref); err == nil {
				data.Payments = append(data.Payments, pr)
			}
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "clients/show.html", data, data.Client.ContactName+" Ledger", "clients")
}

// ShowEditForm renders the Client edit panel.
func (h *ClientsHandler) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientID := chi.URLParam(r, "id")

	var c ClientViewModel
	var companyNameOpt, emailOpt, phoneOpt sql.NullString
	var bStreet, bCity, bState, bZip, bCountry sql.NullString
	var sStreet, sCity, sState, sZip, sCountry sql.NullString
	var preferredCurrency string
	var paymentTerms int
	var whatsapp, website, taxId, category sql.NullString

	query := `
		SELECT 
			id, COALESCE(company_name, ''), contact_name, email, phone, whatsapp, website, tax_id,
			billing_street, billing_city, billing_state, billing_zip, billing_country,
			shipping_street, shipping_city, shipping_state, shipping_zip, shipping_country,
			preferred_currency, payment_terms_days, category, is_active
		FROM clients
		WHERE tenant_id = ? AND id = ?
	`

	var activeInt int
	err := h.app.DB.QueryRowContext(r.Context(), query, tenant.ID, clientID).Scan(
		&c.ID, &companyNameOpt, &c.ContactName, &emailOpt, &phoneOpt, &whatsapp, &website, &taxId,
		&bStreet, &bCity, &bState, &bZip, &bCountry,
		&sStreet, &sCity, &sState, &sZip, &sCountry,
		&preferredCurrency, &paymentTerms, &category, &activeInt,
	)
	c.IsActive = activeInt == 1

	if err != nil {
		SetFlash(w, "flash_error", "Client not found.")
		http.Redirect(w, r, "/clients", http.StatusSeeOther)
		return
	}

	c.CompanyName = companyNameOpt.String
	c.Email = emailOpt.String
	c.Phone = phoneOpt.String

	data := map[string]interface{}{
		"Client":            c,
		"Whatsapp":          whatsapp.String,
		"Website":           website.String,
		"TaxID":             taxId.String,
		"Category":          category.String,
		"PreferredCurrency": preferredCurrency,
		"PaymentTermsDays":  paymentTerms,
		"BillingStreet":     bStreet.String,
		"BillingCity":       bCity.String,
		"BillingState":      bState.String,
		"BillingZip":        bZip.String,
		"BillingCountry":    bCountry.String,
		"ShippingStreet":    sStreet.String,
		"ShippingCity":      sCity.String,
		"ShippingState":     sState.String,
		"ShippingZip":       sZip.String,
		"ShippingCountry":   sCountry.String,
	}

	Render(w, r, h.app.TemplatesFS, "base", "clients/form.html", data, "Edit Client", "clients")
}

// Update edits records of an existing Client in SQLite.
func (h *ClientsHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	contactName := r.FormValue("contact_name")
	if contactName == "" {
		SetFlash(w, "flash_error", "Contact name is required.")
		http.Redirect(w, r, "/clients/"+clientID+"/edit", http.StatusSeeOther)
		return
	}

	now := time.Now().Format(time.RFC3339)

	query := `
		UPDATE clients SET
			company_name = ?, contact_name = ?, email = ?, phone = ?, whatsapp = ?, website = ?, tax_id = ?,
			billing_street = ?, billing_city = ?, billing_state = ?, billing_zip = ?, billing_country = ?,
			shipping_street = ?, shipping_city = ?, shipping_state = ?, shipping_zip = ?, shipping_country = ?,
			preferred_currency = ?, payment_terms_days = ?, category = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		r.FormValue("company_name"),
		contactName,
		r.FormValue("email"),
		r.FormValue("phone"),
		r.FormValue("whatsapp"),
		r.FormValue("website"),
		r.FormValue("tax_id"),
		r.FormValue("billing_street"),
		r.FormValue("billing_city"),
		r.FormValue("billing_state"),
		r.FormValue("billing_zip"),
		r.FormValue("billing_country"),
		r.FormValue("shipping_street"),
		r.FormValue("shipping_city"),
		r.FormValue("shipping_state"),
		r.FormValue("shipping_zip"),
		r.FormValue("shipping_country"),
		r.FormValue("preferred_currency"),
		r.FormValue("payment_terms_days"),
		r.FormValue("category"),
		now,
		tenant.ID, clientID,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/clients/"+clientID+"/edit", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Client profile updated.")
	http.Redirect(w, r, "/clients/"+clientID, http.StatusSeeOther)
}

// Delete soft-deletes a client profile by setting is_active = 0 to prevent integrity cascades.
func (h *ClientsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientID := chi.URLParam(r, "id")

	_, err := h.app.DB.ExecContext(r.Context(), `
		UPDATE clients SET is_active = 0, updated_at = ? WHERE tenant_id = ? AND id = ?
	`, time.Now().Format(time.RFC3339), tenant.ID, clientID)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/clients/"+clientID, http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Client profile successfully deleted.")
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}
