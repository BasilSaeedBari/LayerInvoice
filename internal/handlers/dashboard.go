package handlers

import (
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"net/http"
	"time"
)

// DashboardHandler displays operational stats and summaries.
type DashboardHandler struct {
	app *app.App
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(a *app.App) *DashboardHandler {
	return &DashboardHandler{app: a}
}

// DashboardData structures variables injected into dashboard/index.html.
type DashboardData struct {
	UnpaidInvoicesAmount int64
	UnpaidInvoicesCount  int
	OverdueAmount        int64
	OverdueCount         int
	PendingQuotesAmount  int64
	PendingQuotesCount   int
	RevenueThisMonth     int64
	PrintHoursThisMonth  float64
	FilamentGramsThisMonth float64
	MostUsedFilament     string
	RecentPayments       []RecentPayment
	ActiveClients        []RecentClient
}

type RecentPayment struct {
	ID          string
	Amount      int64
	Currency    string
	PaidAt      string
	CompanyName string
	ContactName string
}

type RecentClient struct {
	ID          string
	CompanyName string
	ContactName string
	Email       string
	Phone       string
}

// Index queries SQLite databases and displays the dashboard interface.
func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	var data DashboardData
	ctx := r.Context()

	// 1. Unpaid Invoices
	err := h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance_due), 0), COUNT(id) 
		FROM invoices 
		WHERE tenant_id = ? AND status IN ('pending', 'sent', 'partially_paid', 'overdue')
	`, tenant.ID).Scan(&data.UnpaidInvoicesAmount, &data.UnpaidInvoicesCount)
	if err != nil {
		h.app.DB.QueryRowContext(ctx, `SELECT 0, 0`).Scan(&data.UnpaidInvoicesAmount, &data.UnpaidInvoicesCount)
	}

	// 2. Overdue Invoices
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance_due), 0), COUNT(id) 
		FROM invoices 
		WHERE tenant_id = ? AND status = 'overdue'
	`, tenant.ID).Scan(&data.OverdueAmount, &data.OverdueCount)
	if err != nil {
		h.app.DB.QueryRowContext(ctx, `SELECT 0, 0`).Scan(&data.OverdueAmount, &data.OverdueCount)
	}

	// 3. Pending Quotes
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total), 0), COUNT(id) 
		FROM quotes 
		WHERE tenant_id = ? AND status IN ('draft', 'sent', 'viewed')
	`, tenant.ID).Scan(&data.PendingQuotesAmount, &data.PendingQuotesCount)
	if err != nil {
		h.app.DB.QueryRowContext(ctx, `SELECT 0, 0`).Scan(&data.PendingQuotesAmount, &data.PendingQuotesCount)
	}

	// 4. Revenue This Month (PKR)
	thisMonth := time.Now().Format("2006-01")
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) 
		FROM payments 
		WHERE tenant_id = ? AND strftime('%Y-%m', paid_at) = ?
	`, tenant.ID, thisMonth).Scan(&data.RevenueThisMonth)

	// 5. Print Hours Billed This Month
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CAST(li.print_hours AS REAL)), 0.0)
		FROM line_items li
		JOIN invoices i ON li.parent_id = i.id
		WHERE li.tenant_id = ? AND li.parent_type = 'invoice' AND li.item_type = '3d_print'
		  AND i.status IN ('paid', 'partially_paid')
		  AND strftime('%Y-%m', i.issue_date) = ?
	`, tenant.ID, thisMonth).Scan(&data.PrintHoursThisMonth)

	// 6. Filament Consumed (Grams) Billed This Month
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CAST(li.print_grams AS REAL)), 0.0)
		FROM line_items li
		JOIN invoices i ON li.parent_id = i.id
		WHERE li.tenant_id = ? AND li.parent_type = 'invoice' AND li.item_type = '3d_print'
		  AND i.status IN ('paid', 'partially_paid')
		  AND strftime('%Y-%m', i.issue_date) = ?
	`, tenant.ID, thisMonth).Scan(&data.FilamentGramsThisMonth)

	// 7. Most Used Filament
	var muf string
	err = h.app.DB.QueryRowContext(ctx, `
		SELECT fp.name || ' (' || fp.material || ')'
		FROM line_items li
		JOIN filament_profiles fp ON li.print_filament_id = fp.id
		WHERE li.tenant_id = ? AND li.item_type = '3d_print'
		GROUP BY fp.id 
		ORDER BY COUNT(li.id) DESC 
		LIMIT 1
	`, tenant.ID).Scan(&muf)
	if err == nil {
		data.MostUsedFilament = muf
	} else {
		data.MostUsedFilament = "None registered yet"
	}

	// 8. Recent Payments
	rows, err := h.app.DB.QueryContext(ctx, `
		SELECT p.id, p.amount, p.currency, p.paid_at, COALESCE(c.company_name, ''), c.contact_name
		FROM payments p
		JOIN clients c ON p.client_id = c.id
		WHERE p.tenant_id = ?
		ORDER BY p.paid_at DESC 
		LIMIT 5
	`, tenant.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rp RecentPayment
			if err := rows.Scan(&rp.ID, &rp.Amount, &rp.Currency, &rp.PaidAt, &rp.CompanyName, &rp.ContactName); err == nil {
				data.RecentPayments = append(data.RecentPayments, rp)
			}
		}
	}

	// 9. Active Clients
	cRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, COALESCE(company_name, ''), contact_name, COALESCE(email, ''), COALESCE(phone, '')
		FROM clients
		WHERE tenant_id = ? AND is_active = 1
		ORDER BY updated_at DESC
		LIMIT 5
	`, tenant.ID)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var rc RecentClient
			if err := cRows.Scan(&rc.ID, &rc.CompanyName, &rc.ContactName, &rc.Email, &rc.Phone); err == nil {
				data.ActiveClients = append(data.ActiveClients, rc)
			}
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "dashboard/index.html", data, "Dashboard", "dashboard")
}
