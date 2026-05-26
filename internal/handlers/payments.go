package handlers

import (
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/db"
	"layerinvoice/internal/money"
	"net/http"
	"time"
)

// PaymentsHandler handles operational cash receipt registers.
type PaymentsHandler struct {
	app *app.App
}

// NewPaymentsHandler creates a new PaymentsHandler.
func NewPaymentsHandler(a *app.App) *PaymentsHandler {
	return &PaymentsHandler{app: a}
}

// PaymentViewModel holds details for list tables.
type PaymentViewModel struct {
	ID             string
	Amount         int64
	Currency       string
	PaidAt         string
	MethodName     string
	TransactionRef string
	Notes          string
	InvoiceNumber  string
	InvoiceID      string
	ClientName     string
	CompanyName    string
}

// PaymentMethodViewModel holds details for payment methods.
type PaymentMethodViewModel struct {
	ID        string
	Name      string
	Type      string
	Details   string
	IsActive  bool
	SortOrder int
}

// PaymentsPageData holds the aggregated lists for list.html.
type PaymentsPageData struct {
	Payments []PaymentViewModel
	Methods  []PaymentMethodViewModel
}

// List queries and displays both paid receipts lists and active billing methods.
func (h *PaymentsHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	var data PaymentsPageData

	// 1. Fetch Recorded Payments
	pQuery := `
		SELECT 
			p.id, p.amount, p.currency, p.paid_at, COALESCE(pm.name, 'Direct Cash'), 
			COALESCE(p.transaction_ref, ''), COALESCE(p.notes, ''),
			i.invoice_number, i.id as invoice_id, c.contact_name, COALESCE(c.company_name, '')
		FROM payments p
		JOIN invoices i ON p.invoice_id = i.id
		JOIN clients c ON p.client_id = c.id
		LEFT JOIN payment_methods pm ON p.payment_method_id = pm.id
		WHERE p.tenant_id = ?
		ORDER BY p.paid_at DESC, p.created_at DESC
	`
	pRows, err := h.app.DB.QueryContext(ctx, pQuery, tenant.ID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pv PaymentViewModel
			err := pRows.Scan(
				&pv.ID, &pv.Amount, &pv.Currency, &pv.PaidAt, &pv.MethodName,
				&pv.TransactionRef, &pv.Notes, &pv.InvoiceNumber, &pv.InvoiceID,
				&pv.ClientName, &pv.CompanyName,
			)
			if err == nil {
				data.Payments = append(data.Payments, pv)
			}
		}
	}

	// 2. Fetch Payment Methods
	mRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, name, type, COALESCE(details, ''), is_active, sort_order
		FROM payment_methods
		WHERE tenant_id = ? AND is_active = 1
		ORDER BY sort_order ASC, name ASC
	`, tenant.ID)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var mv PaymentMethodViewModel
			var activeInt int
			err := mRows.Scan(&mv.ID, &mv.Name, &mv.Type, &mv.Details, &activeInt, &mv.SortOrder)
			if err == nil {
				mv.IsActive = activeInt == 1
				data.Methods = append(data.Methods, mv)
			}
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "payments/list.html", data, "Payments Register", "payments")
}

// ShowNewModal renders the dynamic record payment modal.
func (h *PaymentsHandler) ShowNewModal(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	invoiceID := r.URL.Query().Get("invoice_id")

	if invoiceID == "" {
		w.Write([]byte("Error: invoice_id is required."))
		return
	}

	// Fetch invoice balance, currency, number
	var invNo, currency string
	var balanceDue int64
	err := h.app.DB.QueryRowContext(r.Context(), `
		SELECT invoice_number, balance_due, currency FROM invoices WHERE tenant_id = ? AND id = ?
	`, tenant.ID, invoiceID).Scan(&invNo, &balanceDue, &currency)

	if err != nil {
		w.Write([]byte("Error: invoice not found."))
		return
	}

	// Fetch payment methods
	rows, err := h.app.DB.QueryContext(r.Context(), `
		SELECT id, name FROM payment_methods WHERE tenant_id = ? AND is_active = 1
	`, tenant.ID)
	type MethodOption struct {
		ID   string
		Name string
	}
	var methods []MethodOption
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mo MethodOption
			rows.Scan(&mo.ID, &mo.Name)
			methods = append(methods, mo)
		}
	}

	data := map[string]interface{}{
		"InvoiceID":      invoiceID,
		"InvoiceNumber":  invNo,
		"BalanceDue":     balanceDue,
		"BalanceDisplay": money.FormatAmount(balanceDue),
		"Currency":       currency,
		"Methods":        methods,
		"Today":          time.Now().Format("2006-01-02"),
	}

	RenderPartial(w, r, h.app.TemplatesFS, "templates/payments/new_modal.html", "new_modal", data)
}

// Create inserts a Payment record and updates invoice metrics (amount paid & balance due).
func (h *PaymentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	user, _ := auth.GetUser(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	invoiceID := r.FormValue("invoice_id")
	amountStr := r.FormValue("amount")
	methodID := r.FormValue("payment_method_id")
	ref := r.FormValue("transaction_ref")
	notes := r.FormValue("notes")
	paidAt := r.FormValue("paid_at")

	if invoiceID == "" || amountStr == "" || paidAt == "" {
		SetFlash(w, "flash_error", "Missing required fields to log payment.")
		http.Redirect(w, r, "/invoices", http.StatusSeeOther)
		return
	}

	// Fetch invoice details
	var clientID, currency string
	var total, amountPaid int64
	err := h.app.DB.QueryRowContext(r.Context(), `
		SELECT client_id, currency, total, amount_paid FROM invoices WHERE tenant_id = ? AND id = ?
	`, tenant.ID, invoiceID).Scan(&clientID, &currency, &total, &amountPaid)

	if err != nil {
		SetFlash(w, "flash_error", "Invoice details could not be retrieved.")
		http.Redirect(w, r, "/invoices", http.StatusSeeOther)
		return
	}

	// Parse cash amount
	paymentMoney, err := money.Parse(amountStr, currency)
	if err != nil {
		SetFlash(w, "flash_error", "Invalid payment amount representation.")
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}

	// Start SQLite Transaction
	tx, err := h.app.DB.BeginTx(r.Context(), nil)
	if err != nil {
		SetFlash(w, "flash_error", "Transaction start error.")
		http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// Insert Payment receipt
	paymentID := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	pQuery := `
		INSERT INTO payments (
			id, tenant_id, invoice_id, client_id, amount, currency, payment_method_id,
			transaction_ref, notes, paid_at, created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)
	`

	_, err = tx.ExecContext(r.Context(), pQuery,
		paymentID, tenant.ID, invoiceID, clientID, paymentMoney.Amount, currency,
		methodID, ref, notes, paidAt, user.ID, now,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to record payment: "+err.Error())
		return
	}

	// Compute updated totals
	newAmountPaid := amountPaid + paymentMoney.Amount
	newBalanceDue := total - newAmountPaid

	// Calculate state
	newStatus := "partially_paid"
	if newBalanceDue <= 0 {
		newStatus = "paid"
		newBalanceDue = 0
	}

	// Update Invoice record
	_, err = tx.ExecContext(r.Context(), `
		UPDATE invoices SET
			amount_paid = ?, balance_due = ?, status = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?
	`, newAmountPaid, newBalanceDue, newStatus, now, tenant.ID, invoiceID)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to update invoice metrics.")
		return
	}

	// Commit Transaction
	if err := tx.Commit(); err != nil {
		SetFlash(w, "flash_error", "Failed to commit transaction.")
		return
	}

	SetFlash(w, "flash_success", "Payment receipt successfully recorded!")
	http.Redirect(w, r, "/invoices/"+invoiceID, http.StatusSeeOther)
}

// AddMethod registers a new Payment Method preset.
func (h *PaymentsHandler) AddMethod(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	mType := r.FormValue("type")
	details := r.FormValue("details")

	if name == "" || mType == "" {
		SetFlash(w, "flash_error", "Payment method name and type are required.")
		http.Redirect(w, r, "/payments", http.StatusSeeOther)
		return
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO payment_methods (
			id, tenant_id, name, type, details, is_active, sort_order, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, 0, ?, ?)
	`

	_, err := h.app.DB.ExecContext(r.Context(), query, id, tenant.ID, name, mType, details, now, now)

	if err != nil {
		SetFlash(w, "flash_error", "Failed to register payment method: "+err.Error())
		http.Redirect(w, r, "/payments", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Payment method preset registered successfully.")
	http.Redirect(w, r, "/payments", http.StatusSeeOther)
}
