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
	"layerinvoice/internal/mail"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// InvoicesHandler manages the billing document lifecycle.
type InvoicesHandler struct {
	app                *app.App
	sendDebounceTimers map[string]*time.Timer
	sendDebounceMu     sync.Mutex
}

// NewInvoicesHandler creates a new InvoicesHandler.
func NewInvoicesHandler(a *app.App) *InvoicesHandler {
	return &InvoicesHandler{
		app:                a,
		sendDebounceTimers: make(map[string]*time.Timer),
	}
}

// InvoiceViewModel holds metadata for the main invoices list table.
type InvoiceViewModel struct {
	ID            string
	InvoiceNumber string
	CompanyName   string
	ContactName   string
	IssueDate     string
	DueDate       string
	Total         int64
	BalanceDue    int64
	Status        string
	Currency      string
}

// List displays a table of all invoices.
func (h *InvoicesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	query := `
		SELECT 
			i.id, i.invoice_number, COALESCE(c.company_name, ''), c.contact_name,
			i.issue_date, i.due_date, i.total, i.balance_due, i.status, i.currency
		FROM invoices i
		JOIN clients c ON i.client_id = c.id
		WHERE i.tenant_id = ?
		ORDER BY i.issue_date DESC, i.invoice_number DESC
	`

	rows, err := h.app.DB.QueryContext(r.Context(), query, tenant.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invoices []InvoiceViewModel
	for rows.Next() {
		var iv InvoiceViewModel
		err := rows.Scan(
			&iv.ID, &iv.InvoiceNumber, &iv.CompanyName, &iv.ContactName,
			&iv.IssueDate, &iv.DueDate, &iv.Total, &iv.BalanceDue, &iv.Status, &iv.Currency,
		)
		if err == nil {
			invoices = append(invoices, iv)
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "invoices/list.html", invoices, "Invoices Ledger", "invoices")
}

// ShowNewForm renders the initial client selection form for starting a new invoice.
func (h *InvoicesHandler) ShowNewForm(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	clientIDPrefill := r.URL.Query().Get("client_id")

	// Get all active clients to populate dropdown
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

	// Auto-generate invoice number
	var count int
	h.app.DB.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM invoices WHERE tenant_id = ?", tenant.ID).Scan(&count)
	nextNumber := fmt.Sprintf("INV-%04d", count+1)

	data := map[string]interface{}{
		"Clients":         clients,
		"PrefillClientID": clientIDPrefill,
		"NextNumber":      nextNumber,
		"Today":           time.Now().Format("2006-01-02"),
		"DueDate":         time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
	}

	Render(w, r, h.app.TemplatesFS, "base", "invoices/new.html", data, "Create Invoice", "invoices")
}

// Create inserts a Draft invoice record and redirects to edit/add line items.
func (h *InvoicesHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	user, _ := auth.GetUser(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	invoiceNumber := r.FormValue("invoice_number")
	issueDate := r.FormValue("issue_date")
	dueDate := r.FormValue("due_date")
	currency := r.FormValue("currency")

	if clientID == "" || invoiceNumber == "" || issueDate == "" || dueDate == "" {
		SetFlash(w, "flash_error", "All fields are required to draft an invoice.")
		http.Redirect(w, r, "/invoices/new", http.StatusSeeOther)
		return
	}

	// Confirm invoice number uniqueness
	var exists int
	h.app.DB.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM invoices WHERE tenant_id = ? AND invoice_number = ?", tenant.ID, invoiceNumber).Scan(&exists)
	if exists > 0 {
		SetFlash(w, "flash_error", "Invoice number already exists.")
		http.Redirect(w, r, "/invoices/new", http.StatusSeeOther)
		return
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO invoices (
			id, tenant_id, client_id, invoice_number, status, issue_date, due_date, currency,
			subtotal, total, balance_due, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'draft', ?, ?, ?, 0, 0, 0, ?, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		id, tenant.ID, clientID, invoiceNumber, issueDate, dueDate, currency, user.ID, now, now,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to start draft: "+err.Error())
		http.Redirect(w, r, "/invoices/new", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Invoice draft started successfully.")
	http.Redirect(w, r, "/invoices/"+id, http.StatusSeeOther)
}

// InvoiceDetailsData models the detailed editing layout.
type InvoiceDetailsData struct {
	ID            string
	InvoiceNumber string
	IssueDate     string
	DueDate       string
	Status        string
	Currency      string
	Subtotal      int64
	TaxRate       int64
	TaxAmount     int64
	DiscountValue int64
	Total         int64
	AmountPaid    int64
	BalanceDue    int64
	Notes         string
	Terms         string
	ClientName    string
	CompanyName   string
	ClientEmail   string
	ClientPhone   string
	LineItems     []LineItemViewModel
	Filaments     []FilamentDropdownOption
}

type LineItemViewModel struct {
	ID          string
	Name        string
	Description string
	Quantity    string
	Unit        string
	UnitPrice   int64
	LineTotal   int64
	ItemType    string
}

type FilamentDropdownOption struct {
	ID          string
	Name        string
	CostPerGram int64
}

// Show renders the comprehensive builder screen for a specific invoice.
func (h *InvoicesHandler) Show(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")
	ctx := r.Context()

	var data InvoiceDetailsData
	var compName, notesOpt, termsOpt sql.NullString

	query := `
		SELECT 
			i.id, i.invoice_number, i.issue_date, i.due_date, i.status, i.currency,
			i.subtotal, i.tax_rate, i.tax_amount, i.discount_value, i.total, i.amount_paid, i.balance_due,
			i.notes, i.terms, c.contact_name, COALESCE(c.company_name, ''), COALESCE(c.email, ''), COALESCE(c.phone, '')
		FROM invoices i
		JOIN clients c ON i.client_id = c.id
		WHERE i.tenant_id = ? AND i.id = ?
	`

	err := h.app.DB.QueryRowContext(ctx, query, tenant.ID, invoiceID).Scan(
		&data.ID, &data.InvoiceNumber, &data.IssueDate, &data.DueDate, &data.Status, &data.Currency,
		&data.Subtotal, &data.TaxRate, &data.TaxAmount, &data.DiscountValue, &data.Total, &data.AmountPaid, &data.BalanceDue,
		&notesOpt, &termsOpt, &data.ClientName, &compName, &data.ClientEmail, &data.ClientPhone,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SetFlash(w, "flash_error", "Invoice not found.")
			http.Redirect(w, r, "/invoices", http.StatusSeeOther)
			return
		}
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}

	data.Notes = notesOpt.String
	data.Terms = termsOpt.String
	data.CompanyName = compName.String

	// Load Line Items
	liRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, name, COALESCE(description, ''), quantity, COALESCE(unit, ''), unit_price, line_total, item_type
		FROM line_items
		WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`, tenant.ID, invoiceID)
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

	// Load Filament profiles dropdown if draft mode to support 3D Print modal
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

	Render(w, r, h.app.TemplatesFS, "base", "invoices/show.html", data, data.InvoiceNumber+" Editor", "invoices")
}

// AddCustomRow appends a blank custom line item via HTMX and returns the row snippet.
func (h *InvoicesHandler) AddCustomRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")

	// Ensure editable status (draft or sent)
	status, editable := h.getStatusAndEditable(r.Context(), tenant.ID, invoiceID)
	if !editable {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: invoice is locked."))
		return
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO line_items (
			id, tenant_id, parent_type, parent_id, sort_order, item_type, name, description,
			quantity, unit, unit_price, line_total, created_at, updated_at
		) VALUES (?, ?, 'invoice', ?, 0, 'custom', 'Custom Service / Item', '', '1', 'pcs', 0, 0, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query, id, tenant.ID, invoiceID, now, now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to insert row"))
		return
	}

	// Trigger live recalculation in body
	w.Header().Set("HX-Trigger", "recalculate_totals")

	h.recalculateInvoiceTotals(r.Context(), tenant.ID, invoiceID)

	if status == "sent" {
		h.triggerDebouncedEmail(tenant.ID, invoiceID)
	}

	data := map[string]interface{}{
		"ID":               id,
		"InvoiceID":        invoiceID,
		"Name":             "Custom Service / Item",
		"Description":      "",
		"Quantity":         "1",
		"Unit":             "pcs",
		"UnitPriceDisplay": "0.00",
		"LineTotalDisplay": "0.00",
		"ParentType":       "invoices",
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/invoices/partials/line_item_row.html", "line_item_row", data)
}

// UpdateRow handles inputs adjustments from line item edits and updates totals.
func (h *InvoicesHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")
	liID := chi.URLParam(r, "li_id")

	// Ensure editable status (draft or sent)
	status, editable := h.getStatusAndEditable(r.Context(), tenant.ID, invoiceID)
	if !editable {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: invoice is locked."))
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	qtyStr := r.FormValue("qty")
	priceStr := r.FormValue("unit_price")
	name := r.FormValue("name")
	desc := r.FormValue("description")
	unit := r.FormValue("unit")

	// Parse values
	qtyFloat, err := strconv.ParseFloat(qtyStr, 64)
	if err != nil || qtyFloat < 0 {
		qtyFloat = 1
	}
	qty := calculator.RoundHalfUp(qtyFloat)
	if qty < 1 {
		qty = 1
	}

	priceMoney, err := money.Parse(priceStr, "PKR")
	if err != nil {
		priceMoney = money.Zero("PKR")
	}

	lineTotal := qty * priceMoney.Amount

	// Update DB record
	query := `
		UPDATE line_items SET
			name = ?, description = ?, quantity = ?, unit = ?, unit_price = ?, line_total = ?, updated_at = ?
		WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ? AND id = ?
	`
	now := time.Now().Format(time.RFC3339)
	_, err = h.app.DB.ExecContext(r.Context(), query,
		name, desc, fmt.Sprintf("%d", qty), unit, priceMoney.Amount, lineTotal, now,
		tenant.ID, invoiceID, liID,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Trigger totals recomputation
	h.recalculateInvoiceTotals(r.Context(), tenant.ID, invoiceID)

	if status == "sent" {
		h.triggerDebouncedEmail(tenant.ID, invoiceID)
	}

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(money.FormatAmount(lineTotal)))
}

// DeleteRow removes a line item and updates totals.
func (h *InvoicesHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")
	liID := chi.URLParam(r, "li_id")

	// Ensure editable status (draft or sent)
	status, editable := h.getStatusAndEditable(r.Context(), tenant.ID, invoiceID)
	if !editable {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: invoice is locked."))
		return
	}

	_, err := h.app.DB.ExecContext(r.Context(), `
		DELETE FROM line_items WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ? AND id = ?
	`, tenant.ID, invoiceID, liID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateInvoiceTotals(r.Context(), tenant.ID, invoiceID)

	if status == "sent" {
		h.triggerDebouncedEmail(tenant.ID, invoiceID)
	}

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK) // returns empty body -> HTMX outerHTML swap deletes row on screen!
}

// Totals returns the outerHTML totals block.
func (h *InvoicesHandler) Totals(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")

	var subtotal, taxRate, taxAmount, discountValue, total int64
	var currency string

	err := h.app.DB.QueryRowContext(r.Context(), `
		SELECT subtotal, tax_rate, tax_amount, discount_value, total, currency
		FROM invoices
		WHERE tenant_id = ? AND id = ?
	`, tenant.ID, invoiceID).Scan(&subtotal, &taxRate, &taxAmount, &discountValue, &total, &currency)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"ID":            invoiceID,
		"Currency":      currency,
		"Subtotal":      subtotal,
		"TaxRate":       taxRate,
		"TaxAmount":     taxAmount,
		"DiscountValue": discountValue,
		"Total":         total,
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/invoices/partials/totals.html", "totals", data)
}

// UpdateDiscountTax handles metadata updates from discount/tax basis points additions in the editor.
func (h *InvoicesHandler) UpdateDiscountTax(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")

	// Ensure editable status (draft or sent)
	status, editable := h.getStatusAndEditable(r.Context(), tenant.ID, invoiceID)
	if !editable {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: invoice is locked."))
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	taxStr := r.FormValue("tax_rate")
	discStr := r.FormValue("discount_value")
	notes := r.FormValue("notes")
	terms := r.FormValue("terms")

	// Parse percentage/amount inputs
	taxRate, _ := strconv.ParseInt(taxStr, 10, 64)        // basis points
	discountValue, _ := money.Parse(discStr, "PKR")       // in paisa

	_, err := h.app.DB.ExecContext(r.Context(), `
		UPDATE invoices SET
			tax_rate = ?, discount_value = ?, notes = ?, terms = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?
	`, taxRate, discountValue.Amount, notes, terms, time.Now().Format(time.RFC3339), tenant.ID, invoiceID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.recalculateInvoiceTotals(r.Context(), tenant.ID, invoiceID)

	if status == "sent" {
		h.triggerDebouncedEmail(tenant.ID, invoiceID)
	}

	w.Header().Set("HX-Trigger", "recalculate_totals")
	w.WriteHeader(http.StatusOK)
}

// TransitionStatus shifts the invoice workflow state (e.g. Draft -> Sent / Pending).
func (h *InvoicesHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")
	targetStatus := r.URL.Query().Get("to")

	if targetStatus == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Fetch current status
	var currentStatus string
	err := h.app.DB.QueryRowContext(r.Context(), "SELECT status FROM invoices WHERE tenant_id = ? AND id = ?", tenant.ID, invoiceID).Scan(&currentStatus)
	if err != nil {
		SetFlash(w, "flash_error", "Invoice not found.")
		http.Redirect(w, r, "/invoices", http.StatusSeeOther)
		return
	}

	// Validate status transitions
	allowed := false
	switch currentStatus {
	case "draft":
		allowed = targetStatus == "sent" || targetStatus == "pending"
	case "sent":
		allowed = targetStatus == "sent" || targetStatus == "paid" || targetStatus == "cancelled"
	case "pending":
		allowed = targetStatus == "paid" || targetStatus == "cancelled"
	case "partially_paid":
		allowed = targetStatus == "paid" || targetStatus == "cancelled"
	}

	// Override check for recording payments (handled in payments processor)
	if !allowed && targetStatus != "cancelled" {
		SetFlash(w, "flash_error", "Invalid status workflow transition.")
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	_, err = h.app.DB.ExecContext(r.Context(), `
		UPDATE invoices SET status = ?, updated_at = ? WHERE tenant_id = ? AND id = ?
	`, targetStatus, time.Now().Format(time.RFC3339), tenant.ID, invoiceID)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to transition status.")
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	// Trigger immediate email dispatch if status becomes 'sent'
	if targetStatus == "sent" {
		go h.sendInvoiceEmail(context.Background(), tenant.ID, invoiceID)
	}

	SetFlash(w, "flash_success", "Invoice is now "+targetStatus+".")
	http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
}

// AddPrintItem processes 3D Print Cost modal parameters, creates a print item, and returns row snippet.
func (h *InvoicesHandler) AddPrintItem(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")

	// Ensure editable status (draft or sent)
	status, editable := h.getStatusAndEditable(r.Context(), tenant.ID, invoiceID)
	if !editable {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: invoice is locked."))
		return
	}

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

	// Parse inputs
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

	// Insert row
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
			?, ?, 'invoice', ?, 0, '3d_print', ?, ?,
			'1', 'pcs', ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?
		)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query,
		id, tenant.ID, invoiceID, name, richDesc,
		res.FinalPrice, res.FinalPrice,
		filamentID, fmt.Sprintf("%.2f", grams), res.RawFilamentCost, filamentProfitPct,
		fmt.Sprintf("%.1f", hours), timeRate.Amount, res.TimeCost, labour.Amount, electricity.Amount,
		postproc.Amount, packaging.Amount, shipping.Amount, failureRate,
		profitMultiplier, res.TotalProductionCost, now, now,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Database insertion error: " + err.Error()))
		return
	}

	h.recalculateInvoiceTotals(r.Context(), tenant.ID, invoiceID)

	if status == "sent" {
		h.triggerDebouncedEmail(tenant.ID, invoiceID)
	}

	w.Header().Set("HX-Trigger", "recalculate_totals")

	data := map[string]interface{}{
		"ID":               id,
		"InvoiceID":        invoiceID,
		"Name":             name,
		"Description":      richDesc,
		"Quantity":         "1",
		"Unit":             "pcs",
		"UnitPriceDisplay": money.FormatAmount(res.FinalPrice),
		"LineTotalDisplay": money.FormatAmount(res.FinalPrice),
		"ParentType":       "invoices",
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/invoices/partials/line_item_row.html", "line_item_row", data)
}

// LivePreview computes cost on keypress changes inside the calculator modal.
func (h *InvoicesHandler) LivePreview(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

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

	// Parses
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

	var costPerGram int64
	_ = h.app.DB.QueryRowContext(r.Context(), `
		SELECT cost_per_gram FROM filament_profiles WHERE tenant_id = ? AND id = ?
	`, tenant.ID, filamentID).Scan(&costPerGram)

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

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
		<table style="font-size: var(--text-small); margin-top: var(--space-4);">
			<tbody>
				<tr>
					<td>Raw Filament Cost:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr>
					<td>Filament Profit Markup:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr>
					<td>Print Time Cost:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr>
					<td>Base Production Cost:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr>
					<td>Failure Risk Allowance:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr style="border-top: 1px dashed var(--color-border-strong);">
					<td>Total Production Cost:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr>
					<td>Final Profit Markup:</td>
					<td class="li-col-right li-money">Rs %s</td>
				</tr>
				<tr style="font-weight: 700; border-top: 1px solid var(--color-border-strong);">
					<td>Estimated Customer Price:</td>
					<td class="li-col-right li-money" style="color: var(--color-status-partial); font-size: var(--text-subheading);">Rs %s</td>
				</tr>
			</tbody>
		</table>
	`,
		money.FormatAmount(res.RawFilamentCost),
		money.FormatAmount(res.FilamentProfitMarkup),
		money.FormatAmount(res.TimeCost),
		money.FormatAmount(res.BaseProductionCost),
		money.FormatAmount(res.FailureAllowance),
		money.FormatAmount(res.TotalProductionCost),
		money.FormatAmount(res.FinalProfitMarkup),
		money.FormatAmount(res.FinalPrice),
	)))
}

// Helpers
func (h *InvoicesHandler) recalculateInvoiceTotals(ctx context.Context, tenantID, invoiceID string) {
	// 1. Calculate sum of line item totals
	var subtotal int64
	err := h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(line_total), 0) FROM line_items WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ?
	`, tenantID, invoiceID).Scan(&subtotal)
	if err != nil {
		return
	}

	// 2. Fetch tax rate, discount value, amount paid
	var taxRate, discountValue, amountPaid int64
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT tax_rate, discount_value, amount_paid FROM invoices WHERE tenant_id = ? AND id = ?
	`, tenantID, invoiceID).Scan(&taxRate, &discountValue, &amountPaid)
	if err != nil {
		return
	}

	// 3. Compute totals
	// Discount is simple subtraction
	subAfterDiscount := subtotal - discountValue
	if subAfterDiscount < 0 {
		subAfterDiscount = 0
	}

	// Tax is basis points on subtotal after discount
	taxAmount := calculator.RoundHalfUp(float64(subAfterDiscount) * float64(taxRate) / 10000.0)

	total := subAfterDiscount + taxAmount
	balanceDue := total - amountPaid

	// 4. Update DB
	_, _ = h.app.DB.ExecContext(ctx, `
		UPDATE invoices SET
			subtotal = ?, tax_amount = ?, total = ?, balance_due = ?
		WHERE tenant_id = ? AND id = ?
	`, subtotal, taxAmount, total, balanceDue, tenantID, invoiceID)
}

// getStatusAndEditable returns the current status of the invoice and a boolean indicating if it is editable.
func (h *InvoicesHandler) getStatusAndEditable(ctx context.Context, tenantID, invoiceID string) (string, bool) {
	var status string
	err := h.app.DB.QueryRowContext(ctx, `SELECT status FROM invoices WHERE tenant_id = ? AND id = ?`, tenantID, invoiceID).Scan(&status)
	if err != nil {
		return "", false
	}
	return status, status == "draft" || status == "sent"
}

// generatePDFBytes compiles all metadata, seller settings, and line items, generating raw PDF bytes and returning the invoice number.
func (h *InvoicesHandler) generatePDFBytes(ctx context.Context, tenantID, invoiceID string) ([]byte, string, error) {
	// 1. Fetch Invoice Metadata
	var invNo, issueDate, dueDate, currency, notes, terms string
	var subtotal, taxRate, taxAmount, discountValue, total int64
	var buyerName, buyerCompany, buyerEmail, buyerPhone string
	var bStreet, bCity, bState, bZip, bCountry string

	query := `
		SELECT 
			i.invoice_number, i.issue_date, i.due_date, i.currency, 
			i.subtotal, i.tax_rate, i.tax_amount, i.discount_value, i.total,
			COALESCE(i.notes, ''), COALESCE(i.terms, ''),
			c.contact_name, COALESCE(c.company_name, ''), COALESCE(c.email, ''), COALESCE(c.phone, ''),
			COALESCE(c.billing_street, ''), COALESCE(c.billing_city, ''), COALESCE(c.billing_state, ''), 
			COALESCE(c.billing_zip, ''), COALESCE(c.billing_country, '')
		FROM invoices i
		JOIN clients c ON i.client_id = c.id
		WHERE i.tenant_id = ? AND i.id = ?
	`

	err := h.app.DB.QueryRowContext(ctx, query, tenantID, invoiceID).Scan(
		&invNo, &issueDate, &dueDate, &currency,
		&subtotal, &taxRate, &taxAmount, &discountValue, &total,
		&notes, &terms,
		&buyerName, &buyerCompany, &buyerEmail, &buyerPhone,
		&bStreet, &bCity, &bState, &bZip, &bCountry,
	)

	if err != nil {
		return nil, "", err
	}

	// 2. Fetch Seller Settings
	var tenantName string
	_ = h.app.DB.QueryRowContext(ctx, `SELECT name FROM tenants WHERE id = ?`, tenantID).Scan(&tenantName)
	sellerName := h.getSetting(ctx, tenantID, "company_name", tenantName)
	
	sStreet := h.getSetting(ctx, tenantID, "company_street", "")
	sCity := h.getSetting(ctx, tenantID, "company_city", "")
	sState := h.getSetting(ctx, tenantID, "company_state", "")
	sZip := h.getSetting(ctx, tenantID, "company_zip", "")
	sCountry := h.getSetting(ctx, tenantID, "company_country", "")

	var sellerAddrParts []string
	if sStreet != "" { sellerAddrParts = append(sellerAddrParts, sStreet) }
	if sCity != "" { sellerAddrParts = append(sellerAddrParts, sCity) }
	if sState != "" { sellerAddrParts = append(sellerAddrParts, sState) }
	if sZip != "" { sellerAddrParts = append(sellerAddrParts, sZip) }
	if sCountry != "" { sellerAddrParts = append(sellerAddrParts, sCountry) }
	sellerAddress := strings.Join(sellerAddrParts, ", ")

	sEmail := h.getSetting(ctx, tenantID, "company_email", "")
	sPhone := h.getSetting(ctx, tenantID, "company_phone", "")
	sellerContact := ""
	if sEmail != "" { sellerContact += "Email: " + sEmail }
	if sPhone != "" {
		if sellerContact != "" { sellerContact += " | " }
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
		if buyerContact != "" { buyerContact += " | " }
		buyerContact += "Phone: " + buyerPhone
	}

	// 3. Load Line Items
	rows, err := h.app.DB.QueryContext(ctx, `
		SELECT name, COALESCE(description, ''), quantity, COALESCE(unit, 'pcs'), unit_price, line_total
		FROM line_items
		WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`, tenantID, invoiceID)

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

	companyLogo := h.getSetting(ctx, tenantID, "company_logo", "")
	browserPath := h.getSetting(ctx, tenantID, "pdf_browser_path", "")

	doc := pdf.PDFDocument{
		Title:           "INVOICE",
		DocNumber:       invNo,
		IssueDate:       issueDate,
		ExpiryOrDueDate: dueDate,
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
	return pdfBytes, invNo, err
}

// sendInvoiceEmail is an asynchronous pipeline to mail invoices.
func (h *InvoicesHandler) sendInvoiceEmail(ctx context.Context, tenantID, invoiceID string) {
	smtpHost := h.getSetting(ctx, tenantID, "smtp_host", "")
	smtpPortStr := h.getSetting(ctx, tenantID, "smtp_port", "")
	smtpUser := h.getSetting(ctx, tenantID, "smtp_user", "")
	smtpPass := h.getSetting(ctx, tenantID, "smtp_pass", "")
	smtpFromEmail := h.getSetting(ctx, tenantID, "smtp_from_email", "")
	smtpFromName := h.getSetting(ctx, tenantID, "smtp_from_name", "")

	if smtpHost == "" || smtpPortStr == "" || smtpUser == "" || smtpPass == "" {
		fmt.Printf("[Mailer] SMTP not fully configured for tenant %s. Skipping auto-email.\n", tenantID)
		return
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		fmt.Printf("[Mailer] Invalid SMTP port %s: %v\n", smtpPortStr, err)
		return
	}

	var clientEmail, contactName, invoiceNum, currency string
	var total int64
	query := `
		SELECT c.email, c.contact_name, i.invoice_number, i.currency, i.total
		FROM invoices i
		JOIN clients c ON i.client_id = c.id
		WHERE i.tenant_id = ? AND i.id = ?
	`
	err = h.app.DB.QueryRowContext(ctx, query, tenantID, invoiceID).Scan(
		&clientEmail, &contactName, &invoiceNum, &currency, &total,
	)
	if err != nil {
		fmt.Printf("[Mailer] Error fetching invoice %s details: %v\n", invoiceID, err)
		return
	}

	if clientEmail == "" {
		fmt.Printf("[Mailer] No email on file for client on invoice %s. Skipping email.\n", invoiceNum)
		return
	}

	pdfBytes, _, err := h.generatePDFBytes(ctx, tenantID, invoiceID)
	if err != nil {
		fmt.Printf("[Mailer] Error generating PDF for auto-email: %v\n", err)
		return
	}

	m := mail.NewMailer(smtpHost, smtpPort, smtpUser, smtpPass, smtpFromEmail, smtpFromName)

	formattedTotal := money.New(total, currency).String()
	subject := fmt.Sprintf("Invoice %s from %s", invoiceNum, smtpFromName)
	body := fmt.Sprintf(`Hello %s,

Please find attached your invoice %s for %s.

Thank you for your business!

Best regards,
%s`, contactName, invoiceNum, formattedTotal, smtpFromName)

	bodyHTML := strings.ReplaceAll(body, "\n", "<br>")

	attachment := mail.Attachment{
		Name:    fmt.Sprintf("%s.pdf", invoiceNum),
		Content: pdfBytes,
	}

	err = m.Send(clientEmail, subject, bodyHTML, attachment)

	status := "sent"
	var errStr sql.NullString
	if err != nil {
		status = "failed"
		errStr.String = err.Error()
		errStr.Valid = true
		
		// Structured error log and stderr console display
		slog.Error("SMTP auto-email dispatch failed", "invoice", invoiceNum, "client", clientEmail, "error", err)
		fmt.Printf("[Mailer Error] Failed to send email for invoice %s to %s: %v\n", invoiceNum, clientEmail, err)
	} else {
		slog.Info("SMTP auto-email sent successfully", "invoice", invoiceNum, "client", clientEmail)
		fmt.Printf("[Mailer] Auto-sent invoice %s email successfully to %s\n", invoiceNum, clientEmail)
	}

	// Dynamic database audit log of email dispatch inside email_log
	logID := db.NewULID()
	now := time.Now().Format(time.RFC3339)
	_, logErr := h.app.DB.ExecContext(ctx, `
		INSERT INTO email_log (id, tenant_id, parent_type, parent_id, to_address, subject, status, error, sent_at)
		VALUES (?, ?, 'invoice', ?, ?, ?, ?, ?, ?)
	`, logID, tenantID, invoiceID, clientEmail, subject, status, errStr, now)
	if logErr != nil {
		slog.Error("Failed to save email dispatch log record", "error", logErr)
	}
}

// triggerDebouncedEmail schedules/debounces automatic email dispatch.
func (h *InvoicesHandler) triggerDebouncedEmail(tenantID, invoiceID string) {
	h.sendDebounceMu.Lock()
	defer h.sendDebounceMu.Unlock()

	if t, exists := h.sendDebounceTimers[invoiceID]; exists {
		t.Stop()
	}

	h.sendDebounceTimers[invoiceID] = time.AfterFunc(5*time.Second, func() {
		h.sendDebounceMu.Lock()
		delete(h.sendDebounceTimers, invoiceID)
		h.sendDebounceMu.Unlock()

		h.sendInvoiceEmail(context.Background(), tenantID, invoiceID)
	})
}

// ShowPDF generates the invoice PDF and streams it directly to the browser.
func (h *InvoicesHandler) ShowPDF(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")
	ctx := r.Context()

	pdfBytes, invNo, err := h.generatePDFBytes(ctx, tenant.ID, invoiceID)
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pdf", invNo))
	w.Write(pdfBytes)
}

func (h *InvoicesHandler) getSetting(ctx context.Context, tenantID, key, defaultVal string) string {
	var val string
	err := h.app.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE tenant_id = ? AND key = ?`, tenantID, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

// Delete transactionally deletes an invoice and its associated generic line items. Cascading deletes payments.
func (h *InvoicesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := chi.URLParam(r, "id")

	// 1. Cancel any active auto-email background debounce timers for this invoice
	h.sendDebounceMu.Lock()
	if t, exists := h.sendDebounceTimers[invoiceID]; exists {
		t.Stop()
		delete(h.sendDebounceTimers, invoiceID)
	}
	h.sendDebounceMu.Unlock()

	tx, err := h.app.DB.BeginTx(r.Context(), nil)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to start database transaction: "+err.Error())
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// 2. Delete associated line items
	_, err = tx.ExecContext(r.Context(), `
		DELETE FROM line_items WHERE tenant_id = ? AND parent_type = 'invoice' AND parent_id = ?
	`, tenant.ID, invoiceID)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to delete invoice line items: "+err.Error())
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	// 3. Delete the invoice itself (SQLite ON DELETE CASCADE foreign key automatically deletes associated payments)
	_, err = tx.ExecContext(r.Context(), `
		DELETE FROM invoices WHERE tenant_id = ? AND id = ?
	`, tenant.ID, invoiceID)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to delete invoice: "+err.Error())
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	if err := tx.Commit(); err != nil {
		SetFlash(w, "flash_error", "Transaction commit failed: "+err.Error())
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Invoice successfully deleted.")
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}
