package handlers

import (
	"context"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"net/http"
	"time"
)

// SettingsHandler manages multi-tenant configuration variables.
type SettingsHandler struct {
	app *app.App
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(a *app.App) *SettingsHandler {
	return &SettingsHandler{app: a}
}

// SettingsPageData structures key-value sets for settings forms.
type SettingsPageData struct {
	CompanyName     string
	CompanyStreet   string
	CompanyCity     string
	CompanyState    string
	CompanyZip      string
	CompanyCountry  string
	CompanyEmail    string
	CompanyPhone    string
	CompanyCurrency string

	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPass      string
	SMTPFromEmail string
	SMTPFromName  string
}

// ShowCompany renders the company profile settings panel.
func (h *SettingsHandler) ShowCompany(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	data := SettingsPageData{
		CompanyName:     h.getSetting(ctx, tenant.ID, "company_name", tenant.Name),
		CompanyStreet:   h.getSetting(ctx, tenant.ID, "company_street", ""),
		CompanyCity:     h.getSetting(ctx, tenant.ID, "company_city", ""),
		CompanyState:    h.getSetting(ctx, tenant.ID, "company_state", ""),
		CompanyZip:      h.getSetting(ctx, tenant.ID, "company_zip", ""),
		CompanyCountry:  h.getSetting(ctx, tenant.ID, "company_country", ""),
		CompanyEmail:    h.getSetting(ctx, tenant.ID, "company_email", ""),
		CompanyPhone:    h.getSetting(ctx, tenant.ID, "company_phone", ""),
		CompanyCurrency: h.getSetting(ctx, tenant.ID, "company_currency", "PKR"),
	}

	Render(w, r, h.app.TemplatesFS, "base", "settings/company.html", data, "Company Settings", "settings")
}

// SaveCompany updates company parameters in SQLite.
func (h *SettingsHandler) SaveCompany(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fields := []string{
		"company_name", "company_street", "company_city", "company_state",
		"company_zip", "company_country", "company_email", "company_phone", "company_currency",
	}

	for _, key := range fields {
		val := r.FormValue(key)
		h.saveSetting(ctx, tenant.ID, key, val)
	}

	companyName := r.FormValue("company_name")
	if companyName != "" {
		query := `UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?`
		_, _ = h.app.DB.ExecContext(ctx, query, companyName, time.Now().Format(time.RFC3339), tenant.ID)
	}

	SetFlash(w, "flash_success", "Company profile updated successfully.")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// ShowEmail renders SMTP credentials settings view.
func (h *SettingsHandler) ShowEmail(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	data := SettingsPageData{
		SMTPHost:      h.getSetting(ctx, tenant.ID, "smtp_host", ""),
		SMTPPort:      h.getSetting(ctx, tenant.ID, "smtp_port", "587"),
		SMTPUser:      h.getSetting(ctx, tenant.ID, "smtp_user", ""),
		SMTPPass:      h.getSetting(ctx, tenant.ID, "smtp_pass", ""),
		SMTPFromEmail: h.getSetting(ctx, tenant.ID, "smtp_from_email", ""),
		SMTPFromName:  h.getSetting(ctx, tenant.ID, "smtp_from_name", ""),
	}

	Render(w, r, h.app.TemplatesFS, "base", "settings/email.html", data, "SMTP Server Config", "settings")
}

// SaveEmail records SMTP credentials.
func (h *SettingsHandler) SaveEmail(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fields := []string{
		"smtp_host", "smtp_port", "smtp_user", "smtp_pass", "smtp_from_email", "smtp_from_name",
	}

	for _, key := range fields {
		val := r.FormValue(key)
		h.saveSetting(ctx, tenant.ID, key, val)
	}

	SetFlash(w, "flash_success", "SMTP server configurations saved.")
	http.Redirect(w, r, "/settings/email", http.StatusSeeOther)
}

// Helpers
func (h *SettingsHandler) getSetting(ctx context.Context, tenantID, key, defaultVal string) string {
	var val string
	err := h.app.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE tenant_id = ? AND key = ?`, tenantID, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

func (h *SettingsHandler) saveSetting(ctx context.Context, tenantID, key, value string) {
	query := `
		INSERT INTO settings (tenant_id, key, value) 
		VALUES (?, ?, ?)
		ON CONFLICT(tenant_id, key) DO UPDATE SET value = excluded.value
	`
	_, _ = h.app.DB.ExecContext(ctx, query, tenantID, key, value)
}
