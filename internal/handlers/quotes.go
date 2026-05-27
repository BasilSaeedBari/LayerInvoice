package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/calculator"
	"layerinvoice/internal/db"
	"layerinvoice/internal/money"
	"layerinvoice/internal/pdf"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// QuotesHandler handles billing estimates.
type QuotesHandler struct {
	app *app.App
}

// NewQuotesHandler creates a new QuotesHandler.
func NewQuotesHandler(a *app.App) *QuotesHandler {
	return &QuotesHandler{app: a}
}

// QuoteViewModel holds data for list tables.
type QuoteViewModel struct {
	ID          string
	QuoteNumber string
	CompanyName string
	ContactName string
	IssueDate   string
	ExpiryDate  string
	Total       int64
	Status      string
	Currency    string
}

// List queries and displays all quotes.
func (h *QuotesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	query := `
		SELECT 
			q.id, q.quote_number, COALESCE(c.company_name, ''), c.contact_name,
			q.issue_date, COALESCE(q.expiry_date, ''), q.total, q.status, q.currency
		FROM quotes q
		JOIN clients c ON q.client_id = c.id
		WHERE q.tenant_id = ?
		ORDER BY q.issue_date DESC, q.quote_number DESC
	`

	rows, err := h.app.DB.QueryContext(r.Context(), query, tenant.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var quotes []QuoteViewModel
	for rows.Next() {
		var qv QuoteViewModel
		err := rows.Scan(
			&qv.ID, &qv.QuoteNumber, &qv.CompanyName, &qv.ContactName,
			&qv.IssueDate, &qv.ExpiryDate, &qv.Total, &qv.Status, &qv.Currency,
		)
		if err == nil {
			quotes = append(quotes, qv)
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "quotes/list.html", quotes, "Quotes CRM", "quotes")
}

// ShowNewForm renders client options for starting a quote.
func (h *QuotesHandler) ShowNewForm(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientIDPrefill := r.URL.Query().Get("client_id")

	rows, err := h.app.DB.QueryContext(r.Context(), `
		SELECT id, COALESCE(company_name, ''), contact_name FROM clients WHERE tenant_id = ? AND is_active = 1
	`, tenant.ID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ClientOption struct {
		ID   string
		Name string
	}
	var clients []ClientOption
	for rows.Next() {
		var co ClientOption
		var compName, contName string
		rows.Scan(&co.ID, &compName, &contName)
		if compName != "" {
			co.Name = fmt.Sprintf("%s (%s)", compName, contName)
		} else {
			co.Name = contName
		}
		clients = append(clients, co)
	}

	var count int
	h.app.DB.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM quotes WHERE tenant_id = ?", tenant.ID).Scan(&count)
	nextNumber := fmt.Sprintf("QTE-%04d", count+1)

	data := map[string]interface{}{
		"Clients":         clients,
		"PrefillClientID": clientIDPrefill,
		"NextNumber":      nextNumber,
		"Today":           time.Now().Format("2006-01-02"),
		"ExpiryDate":      time.Now().AddDate(0, 0, 15).Format("2006-01-02"),
	}

	Render(w, r, h.app.TemplatesFS, "base", "quotes/new.html", data, "Create Quote", "quotes")
}

// Create inserts a Quote Draft in SQLite.
func (h *QuotesHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	user, _ := auth.GetUser(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	quoteNumber := r.FormValue("quote_number")
	issueDate := r.FormValue("issue_date")
	expiryDate := r.FormValue("expiry_date")
	currency := r.FormValue("currency")

	if clientID == "" || quoteNumber == "" || issueDate == "" {
		SetFlash(w, "flash_error", "Client selection and quote number are required.")
		http.Redirect(w, r, "/quotes/new", http.StatusSeeOther)
		return
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)
	defaultTerms := h.getSetting(r.Context(), tenant.ID, "default_terms", "")

	query := `
		INSERT INTO quotes (
			id, tenant_id, client_id, quote_number, status, issue_date, expiry_date, currency,
			subtotal, total, terms, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'draft', ?, ?, ?, 0, 0, ?, ?, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		id, tenant.ID, clientID, quoteNumber, issueDate, expiryDate, currency, defaultTerms, user.ID, now, now,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to draft quote: "+err.Error())
		http.Redirect(w, r, "/quotes/new", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Quote draft initiated.")
	http.Redirect(w, r, "/quotes/"+id, http.StatusSeeOther)
}

// QuoteDetailsData models editor contexts.
type QuoteDetailsData struct {
	ID            string
	QuoteNumber   string
	IssueDate     string
	ExpiryDate    string
	Status        string
	Currency      string
	Subtotal      int64
	TaxRate       int64
	TaxAmount     int64
	DiscountValue int64
	Total         int64
	Notes         string
	Terms         string
	ClientName    string
	CompanyName   string
	ClientEmail   string
	ClientPhone   string
	LineItems     []LineItemViewModel
	Filaments     []FilamentDropdownOption
}

// Show renders the Quote workspace and custom calculators.
func (h *QuotesHandler) Show(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")
	ctx := r.Context()

	var data QuoteDetailsData
	var compName, notesOpt, termsOpt, expOpt sql.NullString

	query := `
		SELECT 
			q.id, q.quote_number, q.issue_date, COALESCE(q.expiry_date, ''), q.status, q.currency,
			q.subtotal, q.tax_rate, q.tax_amount, q.discount_value, q.total,
			q.notes, q.terms, c.contact_name, COALESCE(c.company_name, ''), COALESCE(c.email, ''), COALESCE(c.phone, '')
		FROM quotes q
		JOIN clients c ON q.client_id = c.id
		WHERE q.tenant_id = ? AND q.id = ?
	`

	err := h.app.DB.QueryRowContext(ctx, query, tenant.ID, quoteID).Scan(
		&data.ID, &data.QuoteNumber, &data.IssueDate, &expOpt, &data.Status, &data.Currency,
		&data.Subtotal, &data.TaxRate, &data.TaxAmount, &data.DiscountValue, &data.Total,
		&notesOpt, &termsOpt, &data.ClientName, &compName, &data.ClientEmail, &data.ClientPhone,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SetFlash(w, "flash_error", "Quote not found.")
			http.Redirect(w, r, "/quotes", http.StatusSeeOther)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data.ExpiryDate = expOpt.String
	data.Notes = notesOpt.String
	data.Terms = termsOpt.String
	data.CompanyName = compName.String

	// Load Line Items
	liRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, name, COALESCE(description, ''), quantity, COALESCE(unit, ''), unit_price, line_total, item_type
		FROM line_items
		WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`, tenant.ID, quoteID)
	if err == nil {
		defer liRows.Close()
		for liRows.Next() {
			var li LineItemViewModel
			if err := liRows.Scan(&li.ID, &li.Name, &li.Description, &li.Quantity, &li.Unit, &li.UnitPrice, &li.LineTotal, &li.ItemType); err == nil {
				qtyF, _ := strconv.ParseFloat(li.Quantity, 64)
				li.Quantity = fmt.Sprintf("%d", calculator.RoundHalfUp(qtyF))
				data.LineItems = append(data.LineItems, li)
			}
		}
	}

	if data.Status == "draft" {
		fRows, err := h.app.DB.QueryContext(ctx, `
			SELECT id, name, cost_per_gram FROM filament_profiles WHERE tenant_id = ? AND is_active = 1
		`, tenant.ID)
		if err == nil {
			defer fRows.Close()
			for fRows.Next() {
				var fo FilamentDropdownOption
				if err := fRows.Scan(&fo.ID, &fo.Name, &fo.CostPerGram); err == nil {
					data.Filaments = append(data.Filaments, fo)
				}
			}
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "quotes/show.html", data, data.QuoteNumber+" Editor", "quotes")
}

// ConvertToInvoice duplicates Quotes data structures into Invoices data structures in one click.
func (h *QuotesHandler) ConvertToInvoice(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	user, _ := auth.GetUser(r.Context())
	quoteID := chi.URLParam(r, "id")
	ctx := r.Context()

	// 1. Fetch quote
	var clientID, quoteNo, currency string
	var subtotal, taxRate, taxAmount, discountValue, total int64
	var notes, terms sql.NullString
	err := h.app.DB.QueryRowContext(ctx, `
		SELECT client_id, quote_number, currency, subtotal, tax_rate, tax_amount, discount_value, total, notes, terms
		FROM quotes WHERE tenant_id = ? AND id = ?
	`, tenant.ID, quoteID).Scan(&clientID, &quoteNo, &currency, &subtotal, &taxRate, &taxAmount, &discountValue, &total, &notes, &terms)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to retrieve quote details.")
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	// 2. Auto-generate invoice number
	var count int
	h.app.DB.QueryRowContext(ctx, "SELECT COUNT(id) FROM invoices WHERE tenant_id = ?", tenant.ID).Scan(&count)
	invNumber := fmt.Sprintf("INV-%s", quoteNo[4:]) // match quote number suffix or simple count

	invoiceID := db.NewULID()
	today := time.Now().Format("2006-01-02")
	dueDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	now := time.Now().Format(time.RFC3339)

	// Start database transaction to ensure atomicity
	tx, err := h.app.DB.BeginTx(ctx, nil)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to start transaction: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// 3. Create Invoice
	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoices (
			id, tenant_id, client_id, quote_id, invoice_number, status, issue_date, due_date, currency,
			subtotal, discount_type, discount_value, tax_rate, tax_amount, total, amount_paid, balance_due,
			notes, terms, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 'draft', ?, ?, ?, ?, 'fixed', ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)
	`, invoiceID, tenant.ID, clientID, quoteID, invNumber, today, dueDate, currency, subtotal, discountValue, taxRate, taxAmount, total, total, notes, terms, user.ID, now, now)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to insert invoice: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	// 4. Duplicate Line Items
	rows, err := tx.QueryContext(ctx, `
		SELECT 
			item_type, name, description, quantity, unit, unit_price, discount_type, discount_value, tax_rate, tax_amount, line_total,
			print_filament_id, print_grams, print_filament_cost, print_filament_profit_pct, print_hours, print_time_rate,
			print_time_cost, print_labour_cost, print_electricity_cost, print_postproc_cost, print_packaging_cost,
			print_shipping_cost, print_failure_rate, print_profit_multiplier, print_production_cost
		FROM line_items
		WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ?
	`, tenant.ID, quoteID)

	if err == nil {
		defer rows.Close()
		var itemsToInsert [][]interface{}
		for rows.Next() {
			var itemType, name, desc, qty, unit, discType string
			var uPrice, discVal, tRate, tAmt, lTotal int64
			var printFilID sql.NullString
			var printGrams, printHours sql.NullString
			var pFilCost, pFilProfPct, pTimeRate, pTimeCost, pLab, pElec, pPost, pPkg, pShip, pFail, pMult, pProd sql.NullInt64

			err := rows.Scan(
				&itemType, &name, &desc, &qty, &unit, &uPrice, &discType, &discVal, &tRate, &tAmt, &lTotal,
				&printFilID, &printGrams, &pFilCost, &pFilProfPct, &printHours, &pTimeRate,
				&pTimeCost, &pLab, &pElec, &pPost, &pPkg, &pShip, &pFail, &pMult, &pProd,
			)

			if err == nil {
				newLIID := db.NewULID()
				itemsToInsert = append(itemsToInsert, []interface{}{
					newLIID, tenant.ID, invoiceID, itemType, name, desc, qty, unit, uPrice, discType, discVal, tRate, tAmt, lTotal,
					printFilID.String, printGrams.String, pFilCost.Int64, pFilProfPct.Int64, printHours.String, pTimeRate.Int64,
					pTimeCost.Int64, pLab.Int64, pElec.Int64, pPost.Int64, pPkg.Int64, pShip.Int64, pFail.Int64, pMult.Int64, pProd.Int64,
				})
			}
		}

		for _, item := range itemsToInsert {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO line_items (
					id, tenant_id, parent_type, parent_id, sort_order, item_type, name, description,
					quantity, unit, unit_price, discount_type, discount_value, tax_rate, tax_amount, line_total,
					print_filament_id, print_grams, print_filament_cost, print_filament_profit_pct, print_hours, print_time_rate,
					print_time_cost, print_labour_cost, print_electricity_cost, print_postproc_cost, print_packaging_cost,
					print_shipping_cost, print_failure_rate, print_profit_multiplier, print_production_cost, created_at, updated_at
				) VALUES (
					?, ?, 'invoice', ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
					NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, ''), NULLIF(?, 0),
					NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0),
					NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), ?, ?
				)
			`, append(item, now, now)...)
			if err != nil {
				SetFlash(w, "flash_error", "Failed to duplicate line item: "+err.Error())
				http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
				return
			}
		}
	}

	// 5. Update Quote status & link
	_, err = tx.ExecContext(ctx, `
		UPDATE quotes SET status = 'converted', invoice_id = ?, updated_at = ? WHERE tenant_id = ? AND id = ?
	`, invoiceID, now, tenant.ID, quoteID)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to update quote status.")
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	// Commit Transaction
	if err := tx.Commit(); err != nil {
		SetFlash(w, "flash_error", "Failed to commit transaction.")
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Quote converted to Invoice draft successfully.")
	http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
}

// Duplicates core functions of Invoices for Quotes compatibility
func (h *QuotesHandler) AddCustomRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO line_items (
			id, tenant_id, parent_type, parent_id, sort_order, item_type, name, description,
			quantity, unit, unit_price, line_total, created_at, updated_at
		) VALUES (?, ?, 'quote', ?, 0, 'custom', 'Custom Estimate Item', '', '1', 'pcs', 0, 0, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query, id, tenant.ID, quoteID, now, now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "recalculate_totals")

	data := map[string]interface{}{
		"ID":               id,
		"InvoiceID":        quoteID, // matches HTML input field names
		"Name":             "Custom Estimate Item",
		"Description":      "",
		"Quantity":         "1",
		"Unit":             "pcs",
		"UnitPriceDisplay": "0.00",
		"LineTotalDisplay": "0.00",
		"ParentType":       "quotes",
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/invoices/partials/line_item_row.html", "line_item_row", data)
}

func (h *QuotesHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")
	liID := chi.URLParam(r, "li_id")

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	qtyStr := r.FormValue("qty")
	priceStr := r.FormValue("unit_price")
	name := r.FormValue("name")
	desc := r.FormValue("description")
	unit := r.FormValue("unit")

	qtyFloat, _ := strconv.ParseFloat(qtyStr, 64)
	qty := calculator.RoundHalfUp(qtyFloat)
	if qty < 1 {
		qty = 1
	}
	priceMoney, _ := money.Parse(priceStr, "PKR")
	lineTotal := qty * priceMoney.Amount

	query := `
		UPDATE line_items SET
			name = ?, description = ?, quantity = ?, unit = ?, unit_price = ?, line_total = ?, updated_at = ?
		WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ? AND id = ?
	`
	_, err := h.app.DB.ExecContext(r.Context(), query,
		name, desc, fmt.Sprintf("%d", qty), unit, priceMoney.Amount, lineTotal, time.Now().Format(time.RFC3339),
		tenant.ID, quoteID, liID,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateQuoteTotals(r.Context(), tenant.ID, quoteID)

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(money.FormatAmount(lineTotal)))
}

func (h *QuotesHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")
	liID := chi.URLParam(r, "li_id")

	_, err := h.app.DB.ExecContext(r.Context(), `
		DELETE FROM line_items WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ? AND id = ?
	`, tenant.ID, quoteID, liID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateQuoteTotals(r.Context(), tenant.ID, quoteID)

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK)
}

func (h *QuotesHandler) Totals(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")

	var subtotal, taxRate, taxAmount, discountValue, total int64
	var currency string

	err := h.app.DB.QueryRowContext(r.Context(), `
		SELECT subtotal, tax_rate, tax_amount, discount_value, total, currency
		FROM quotes
		WHERE tenant_id = ? AND id = ?
	`, tenant.ID, quoteID).Scan(&subtotal, &taxRate, &taxAmount, &discountValue, &total, &currency)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"ID":            quoteID,
		"Currency":      currency,
		"Subtotal":      subtotal,
		"TaxRate":       taxRate,
		"TaxAmount":     taxAmount,
		"DiscountValue": discountValue,
		"Total":         total,
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/quotes/partials/totals.html", "totals", data)
}

func (h *QuotesHandler) UpdateDiscountTax(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	taxStr := r.FormValue("tax_rate")
	discStr := r.FormValue("discount_value")
	notes := r.FormValue("notes")
	terms := r.FormValue("terms")

	taxRate, _ := strconv.ParseInt(taxStr, 10, 64)
	discountValue, _ := money.Parse(discStr, "PKR")

	_, err := h.app.DB.ExecContext(r.Context(), `
		UPDATE quotes SET
			tax_rate = ?, discount_value = ?, notes = ?, terms = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?
	`, taxRate, discountValue.Amount, notes, terms, time.Now().Format(time.RFC3339), tenant.ID, quoteID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateQuoteTotals(r.Context(), tenant.ID, quoteID)

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK)
}

func (h *QuotesHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")
	targetStatus := r.URL.Query().Get("to")

	_, err := h.app.DB.ExecContext(r.Context(), `
		UPDATE quotes SET status = ?, updated_at = ? WHERE tenant_id = ? AND id = ?
	`, targetStatus, time.Now().Format(time.RFC3339), tenant.ID, quoteID)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to update estimate status.")
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Quote is now "+targetStatus+".")
	http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
}

func (h *QuotesHandler) AddPrintItem(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filamentID := r.FormValue("filament_id")
	gramsStr := r.FormValue("grams")
	filamentProfitStr := r.FormValue("filament_profit_pct")
	hoursStr := r.FormValue("print_hours")
	timeRateStr := r.FormValue("time_rate")
	labourStr := r.FormValue("labour_cost")
	elecStr := r.FormValue("electricity_cost")
	postStr := r.FormValue("postproc_cost")
	pkgStr := r.FormValue("packaging_cost")
	shipStr := r.FormValue("shipping_cost")
	failStr := r.FormValue("failure_rate")
	multStr := r.FormValue("profit_multiplier")
	name := r.FormValue("print_name")
	description := r.FormValue("print_desc")

	grams, _ := strconv.ParseFloat(gramsStr, 64)
	filamentProfitPct, _ := strconv.ParseInt(filamentProfitStr, 10, 64)
	hours, _ := strconv.ParseFloat(hoursStr, 64)
	timeRate, _ := money.Parse(timeRateStr, "PKR")
	labour, _ := money.Parse(labourStr, "PKR")
	electricity, _ := money.Parse(elecStr, "PKR")
	postproc, _ := money.Parse(postStr, "PKR")
	packaging, _ := money.Parse(pkgStr, "PKR")
	shipping, _ := money.Parse(shipStr, "PKR")
	failureRate, _ := strconv.ParseInt(failStr, 10, 64)
	profitMultiplier, _ := strconv.ParseInt(multStr, 10, 64)

	// Fetch filament cost per gram and details
	var costPerGram int64
	var filName, filBrand, filMaterial string
	_ = h.app.DB.QueryRowContext(r.Context(), `
		SELECT cost_per_gram, name, COALESCE(brand, ''), material 
		FROM filament_profiles 
		WHERE tenant_id = ? AND id = ?
	`, tenant.ID, filamentID).Scan(&costPerGram, &filName, &filBrand, &filMaterial)

	// Calculate cost
	calcIn := calculator.PrintJobInputs{
		Grams:             grams,
		CostPerGram:       costPerGram,
		FilamentProfitPct: filamentProfitPct,
		PrintHours:        hours,
		TimeRate:          timeRate.Amount,
		LabourCost:        labour.Amount,
		ElectricityCost:   electricity.Amount,
		PostprocCost:      postproc.Amount,
		PackagingCost:     packaging.Amount,
		ShippingCost:      shipping.Amount,
		FailureRate:       failureRate,
		ProfitMultiplier:  profitMultiplier,
	}

	res := calculator.Calculate(calcIn)

	// Build dynamic rich sub-description
	richDesc := ""
	if description != "" {
		richDesc += description + "\n"
	}
	filInfo := filName
	if filBrand != "" {
		filInfo = filBrand + " " + filInfo
	}
	if filMaterial != "" {
		filInfo = filInfo + " (" + filMaterial + ")"
	}
	richDesc += fmt.Sprintf("• Filament: %s [%.2fg used]\n", filInfo, grams)
	richDesc += fmt.Sprintf("• Print Time: %.1f hours (Rate: Rs %.2f/hr)\n", hours, float64(timeRate.Amount)/100.0)

	var overheads []string
	if labour.Amount > 0 {
		overheads = append(overheads, fmt.Sprintf("Labor (Rs %.2f)", float64(labour.Amount)/100.0))
	}
	if electricity.Amount > 0 {
		overheads = append(overheads, fmt.Sprintf("Electricity (Rs %.2f)", float64(electricity.Amount)/100.0))
	}
	if postproc.Amount > 0 {
		overheads = append(overheads, fmt.Sprintf("Post-processing (Rs %.2f)", float64(postproc.Amount)/100.0))
	}
	if packaging.Amount > 0 {
		overheads = append(overheads, fmt.Sprintf("Packaging (Rs %.2f)", float64(packaging.Amount)/100.0))
	}
	if shipping.Amount > 0 {
		overheads = append(overheads, fmt.Sprintf("Shipping (Rs %.2f)", float64(shipping.Amount)/100.0))
	}

	if len(overheads) > 0 {
		richDesc += "• Overheads: " + strings.Join(overheads, ", ")
	} else {
		richDesc = strings.TrimSuffix(richDesc, "\n")
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO line_items (
			id, tenant_id, parent_type, parent_id, sort_order, item_type, name, description,
			quantity, unit, unit_price, line_total,
			print_filament_id, print_grams, print_filament_cost, print_filament_profit_pct,
			print_hours, print_time_rate, print_time_cost, print_labour_cost, print_electricity_cost,
			print_postproc_cost, print_packaging_cost, print_shipping_cost, print_failure_rate,
			print_profit_multiplier, print_production_cost, created_at, updated_at
		) VALUES (
			?, ?, 'quote', ?, 0, '3d_print', ?, ?,
			'1', 'pcs', ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?
		)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		id, tenant.ID, quoteID, name, richDesc,
		res.FinalPrice, res.FinalPrice,
		filamentID, fmt.Sprintf("%.2f", grams), res.RawFilamentCost, filamentProfitPct,
		fmt.Sprintf("%.1f", hours), timeRate.Amount, res.TimeCost, labour.Amount, electricity.Amount,
		postproc.Amount, packaging.Amount, shipping.Amount, failureRate,
		profitMultiplier, res.TotalProductionCost, now, now,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateQuoteTotals(r.Context(), tenant.ID, quoteID)

	w.Header().Set("HX-Trigger", "recalculate_totals")

	data := map[string]interface{}{
		"ID":               id,
		"InvoiceID":        quoteID,
		"Name":             name,
		"Description":      richDesc,
		"Quantity":         "1",
		"Unit":             "pcs",
		"UnitPriceDisplay": money.FormatAmount(res.FinalPrice),
		"LineTotalDisplay": money.FormatAmount(res.FinalPrice),
		"ParentType":       "quotes",
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/invoices/partials/line_item_row.html", "line_item_row", data)
}

func (h *QuotesHandler) recalculateQuoteTotals(ctx context.Context, tenantID, quoteID string) {
	var subtotal int64
	err := h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(line_total), 0) FROM line_items WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ?
	`, tenantID, quoteID).Scan(&subtotal)
	if err != nil {
		return
	}

	var taxRate, discountValue int64
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT tax_rate, discount_value FROM quotes WHERE tenant_id = ? AND id = ?
	`, tenantID, quoteID).Scan(&taxRate, &discountValue)
	if err != nil {
		return
	}

	subAfterDiscount := subtotal - discountValue
	if subAfterDiscount < 0 {
		subAfterDiscount = 0
	}

	taxAmount := calculator.RoundHalfUp(float64(subAfterDiscount) * float64(taxRate) / 10000.0)
	total := subAfterDiscount + taxAmount

	_, _ = h.app.DB.ExecContext(ctx, `
		UPDATE quotes SET
			subtotal = ?, tax_amount = ?, total = ?
		WHERE tenant_id = ? AND id = ?
	`, subtotal, taxAmount, total, tenantID, quoteID)
}

// ShowPDF generates the quote PDF and streams it directly to the browser.
func (h *QuotesHandler) ShowPDF(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")
	ctx := r.Context()

	// 1. Fetch Quote Metadata
	var quoteNo, issueDate, expiryDate, currency, notes, terms string
	var subtotal, taxRate, taxAmount, discountValue, total int64
	var buyerName, buyerCompany, buyerEmail, buyerPhone string
	var bStreet, bCity, bState, bZip, bCountry string

	query := `
		SELECT 
			q.quote_number, q.issue_date, COALESCE(q.expiry_date, ''), q.currency, 
			q.subtotal, q.tax_rate, q.tax_amount, q.discount_value, q.total,
			COALESCE(q.notes, ''), COALESCE(q.terms, ''),
			c.contact_name, COALESCE(c.company_name, ''), COALESCE(c.email, ''), COALESCE(c.phone, ''),
			COALESCE(c.billing_street, ''), COALESCE(c.billing_city, ''), COALESCE(c.billing_state, ''), 
			COALESCE(c.billing_zip, ''), COALESCE(c.billing_country, '')
		FROM quotes q
		JOIN clients c ON q.client_id = c.id
		WHERE q.tenant_id = ? AND q.id = ?
	`

	err := h.app.DB.QueryRowContext(ctx, query, tenant.ID, quoteID).Scan(
		&quoteNo, &issueDate, &expiryDate, &currency,
		&subtotal, &taxRate, &taxAmount, &discountValue, &total,
		&notes, &terms,
		&buyerName, &buyerCompany, &buyerEmail, &buyerPhone,
		&bStreet, &bCity, &bState, &bZip, &bCountry,
	)

	if err != nil {
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	// 2. Fetch Seller Settings
	sellerName := h.getSetting(ctx, tenant.ID, "company_name", tenant.Name)
	
	sStreet := h.getSetting(ctx, tenant.ID, "company_street", "")
	sCity := h.getSetting(ctx, tenant.ID, "company_city", "")
	sState := h.getSetting(ctx, tenant.ID, "company_state", "")
	sZip := h.getSetting(ctx, tenant.ID, "company_zip", "")
	sCountry := h.getSetting(ctx, tenant.ID, "company_country", "")

	var sellerAddrParts []string
	if sStreet != "" { sellerAddrParts = append(sellerAddrParts, sStreet) }
	if sCity != "" { sellerAddrParts = append(sellerAddrParts, sCity) }
	if sState != "" { sellerAddrParts = append(sellerAddrParts, sState) }
	if sZip != "" { sellerAddrParts = append(sellerAddrParts, sZip) }
	if sCountry != "" { sellerAddrParts = append(sellerAddrParts, sCountry) }
	sellerAddress := strings.Join(sellerAddrParts, ", ")

	sEmail := h.getSetting(ctx, tenant.ID, "company_email", "")
	sPhone := h.getSetting(ctx, tenant.ID, "company_phone", "")
	sellerContact := ""
	if sEmail != "" { sellerContact += "Email: " + sEmail }
	if sPhone != "" {
		if sellerContact != "" { sellerContact += "\n" }
		sellerContact += "Phone: " + sPhone
	}

	// Buyer Address
	var buyerAddrParts []string
	if bStreet != "" { buyerAddrParts = append(buyerAddrParts, bStreet) }
	if bCity != "" { buyerAddrParts = append(buyerAddrParts, bCity) }
	if bState != "" { buyerAddrParts = append(buyerAddrParts, bState) }
	if bZip != "" { buyerAddrParts = append(buyerAddrParts, bZip) }
	if bCountry != "" { buyerAddrParts = append(buyerAddrParts, bCountry) }
	buyerAddress := strings.Join(buyerAddrParts, ", ")

	buyerContact := ""
	if buyerEmail != "" { buyerContact += "Email: " + buyerEmail }
	if buyerPhone != "" {
		if buyerContact != "" { buyerContact += "\n" }
		buyerContact += "Phone: " + buyerPhone
	}

	// 3. Load Line Items
	rows, err := h.app.DB.QueryContext(ctx, `
		SELECT name, COALESCE(description, ''), quantity, COALESCE(unit, 'pcs'), unit_price, line_total
		FROM line_items
		WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`, tenant.ID, quoteID)

	var items []pdf.PDFLineItem
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var li pdf.PDFLineItem
			var qtyStr string
			rows.Scan(&li.Name, &li.Description, &qtyStr, &li.Unit, &li.UnitPrice, &li.LineTotal)
			
			qty, _ := strconv.ParseFloat(qtyStr, 64)
			li.Quantity = fmt.Sprintf("%d", calculator.RoundHalfUp(qty))
			
			items = append(items, li)
		}
	}

	companyLogo := h.getSetting(ctx, tenant.ID, "company_logo", "")
	browserPath := h.getSetting(ctx, tenant.ID, "pdf_browser_path", "")

	doc := pdf.PDFDocument{
		Title:           "QUOTE",
		DocNumber:       quoteNo,
		IssueDate:       issueDate,
		ExpiryOrDueDate: expiryDate,
		Currency:        currency,
		CompanyLogo:     companyLogo,
		SellerName:      sellerName,
		SellerAddress:   sellerAddress,
		SellerContact:   sellerContact,
		BuyerName:       buyerName,
		BuyerCompany:    buyerCompany,
		BuyerAddress:    buyerAddress,
		BuyerContact:    buyerContact,
		LineItems:       items,
		Subtotal:        subtotal,
		TaxRate:         taxRate,
		TaxAmount:       taxAmount,
		DiscountValue:   discountValue,
		Total:           total,
		Notes:           notes,
		Terms:           terms,
	}

	pdfBytes, err := pdf.Generate(doc, browserPath)
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pdf", quoteNo))
	w.Write(pdfBytes)
}

func (h *QuotesHandler) getSetting(ctx context.Context, tenantID, key, defaultVal string) string {
	var val string
	err := h.app.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE tenant_id = ? AND key = ?`, tenantID, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

// Delete transactionally deletes a quote and its associated generic line items.
func (h *QuotesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	quoteID := chi.URLParam(r, "id")

	tx, err := h.app.DB.BeginTx(r.Context(), nil)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to start database transaction: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// 1. Delete associated line items
	_, err = tx.ExecContext(r.Context(), `
		DELETE FROM line_items WHERE tenant_id = ? AND parent_type = 'quote' AND parent_id = ?
	`, tenant.ID, quoteID)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to delete quote line items: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	// 2. Delete the quote record
	_, err = tx.ExecContext(r.Context(), `
		DELETE FROM quotes WHERE tenant_id = ? AND id = ?
	`, tenant.ID, quoteID)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to delete quote: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	if err := tx.Commit(); err != nil {
		SetFlash(w, "flash_error", "Transaction commit failed: "+err.Error())
		http.Redirect(w, r, "/quotes/"+quoteID, http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Quote successfully deleted.")
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}
