package handlers

import (
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/db"
	"net/http"
	"strconv"
	"time"
)

// SystemHandler coordinates administrators access, tax settings, and `/setup` bootstrapping.
type SystemHandler struct {
	app *app.App
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler(a *app.App) *SystemHandler {
	return &SystemHandler{app: a}
}

// SystemPageData holds listings for system management.
type SystemPageData struct {
	Users    []UserViewModel
	TaxRates []TaxRateViewModel
}

type UserViewModel struct {
	ID        string
	Name      string
	Email     string
	Role      string
	IsActive  bool
	CreatedAt string
}

type TaxRateViewModel struct {
	ID        string
	Name      string
	Rate      int64 // basis points
	IsDefault bool
}

// ShowSystem renders system users and tax configurations.
func (h *SystemHandler) ShowSystem(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	var data SystemPageData

	// 1. Fetch Users
	rows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, name, email, role, is_active, created_at
		FROM users
		WHERE tenant_id = ?
		ORDER BY role ASC, name ASC
	`, tenant.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var u UserViewModel
			var activeInt int
			err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &activeInt, &u.CreatedAt)
			if err == nil {
				u.IsActive = activeInt == 1
				data.Users = append(data.Users, u)
			}
		}
	}

	// 2. Fetch Tax Rates
	tRows, err := h.app.DB.QueryContext(ctx, `
		SELECT id, name, rate, is_default
		FROM tax_rates
		WHERE tenant_id = ? AND is_active = 1
		ORDER BY is_default DESC, name ASC
	`, tenant.ID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var tr TaxRateViewModel
			var defInt int
			err := tRows.Scan(&tr.ID, &tr.Name, &tr.Rate, &defInt)
			if err == nil {
				tr.IsDefault = defInt == 1
				data.TaxRates = append(data.TaxRates, tr)
			}
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "system/index.html", data, "System Controls", "system")
}

// CreateUser registers a new system administrator.
func (h *SystemHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	role := r.FormValue("role")

	if name == "" || email == "" || password == "" || role == "" {
		SetFlash(w, "flash_error", "All fields are required to register a user.")
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	// Hash password
	hash, err := auth.HashPassword(password)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to hash password.")
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	userID := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO users (id, tenant_id, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
	`

	_, err = h.app.DB.ExecContext(r.Context(), query, userID, tenant.ID, name, email, hash, role, now, now)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "System user "+name+" registered successfully.")
	http.Redirect(w, r, "/system", http.StatusSeeOther)
}

// CreateTaxRate creates a new tax rate preset.
func (h *SystemHandler) CreateTaxRate(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	rateStr := r.FormValue("rate")
	isDefaultStr := r.FormValue("is_default")

	if name == "" || rateStr == "" {
		SetFlash(w, "flash_error", "Tax rate name and percentage are required.")
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	rate, err := strconv.ParseInt(rateStr, 10, 64)
	if err != nil {
		SetFlash(w, "flash_error", "Invalid tax rate (must be basis points).")
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	isDefault := 0
	if isDefaultStr == "1" {
		isDefault = 1
		// Reset other defaults for this tenant
		_, _ = h.app.DB.ExecContext(r.Context(), `UPDATE tax_rates SET is_default = 0 WHERE tenant_id = ?`, tenant.ID)
	}

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO tax_rates (id, tenant_id, name, rate, is_default, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`

	_, err = h.app.DB.ExecContext(r.Context(), query, id, tenant.ID, name, rate, isDefault, now, now)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/system", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Tax rate preset created successfully.")
	http.Redirect(w, r, "/system", http.StatusSeeOther)
}

// ShowSetup renders the `/setup` bootstrap screen.
func (h *SystemHandler) ShowSetup(w http.ResponseWriter, r *http.Request) {
	// If any user exists, redirect to login page for safety!
	var exists int
	err := h.app.DB.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM users").Scan(&exists)
	if err == nil && exists > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Render layout-free bootstrap screen
	Render(w, r, h.app.TemplatesFS, "auth", "system/setup.html", nil, "System Setup", "")
}

// HandleSetup bootstraps the first multi-tenant Company (Tenant) and Owner.
func (h *SystemHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	// If any user exists, forbid setup!
	var exists int
	err := h.app.DB.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM users").Scan(&exists)
	if err == nil && exists > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	companyName := r.FormValue("company_name")
	companySlug := r.FormValue("company_slug")
	adminName := r.FormValue("admin_name")
	adminEmail := r.FormValue("admin_email")
	password := r.FormValue("password")

	if companyName == "" || companySlug == "" || adminName == "" || adminEmail == "" || password == "" {
		SetFlash(w, "flash_error", "All fields are required to bootstrap the system.")
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	// Hash password
	hash, err := auth.HashPassword(password)
	if err != nil {
		SetFlash(w, "flash_error", "Password hashing failed.")
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	tenantID := db.NewULID()
	adminID := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	// Execute Setup inside transaction
	tx, err := h.app.DB.BeginTx(r.Context(), nil)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to start transaction.")
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// 1. Create Tenant
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO tenants (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
	`, tenantID, companyName, companySlug, now, now)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to insert company: "+err.Error())
		return
	}

	// 2. Create Owner User
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO users (id, tenant_id, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'owner', 1, ?, ?)
	`, adminID, tenantID, adminName, adminEmail, hash, now, now)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to insert owner account: "+err.Error())
		return
	}

	// 3. Seed initial default bank transfer payment method
	pmID := db.NewULID()
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO payment_methods (id, tenant_id, name, type, details, is_active, sort_order, created_at, updated_at)
		VALUES (?, ?, 'Direct Cash', 'cash', 'Cash hand-in / in-person payment', 1, 0, ?, ?)
	`, pmID, tenantID, now, now)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to register default payment method.")
		return
	}

	// Commit setup
	if err := tx.Commit(); err != nil {
		SetFlash(w, "flash_error", "Failed to commit transaction.")
		return
	}

	// 4. Create Session and Log In Owner immediately!
	store := auth.NewSessionStore(h.app.DB.DB)
	sessionID, err := store.CreateSession(r.Context(), adminID, 24*time.Hour)
	if err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "layerinvoice_session",
			Value:    sessionID,
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		})
	}

	SetFlash(w, "flash_success", "System bootstrapped successfully! Welcome to LayerInvoice.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
