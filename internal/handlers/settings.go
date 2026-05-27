package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/mail"
	"log/slog"
	"net/http"
	"strconv"
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
	CompanyLogo     string
	DefaultTerms    string

	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPass      string
	SMTPFromEmail string
	SMTPFromName  string
	SMTPBccEmail  string
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
		CompanyLogo:     h.getSetting(ctx, tenant.ID, "company_logo", ""),
		DefaultTerms:    h.getSetting(ctx, tenant.ID, "default_terms", ""),
	}

	Render(w, r, h.app.TemplatesFS, "base", "settings/company.html", data, "Company Settings", "settings")
}

// SaveCompany updates company parameters in SQLite.
func (h *SettingsHandler) SaveCompany(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	// Parse multipart form (max 5MB file upload)
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		// Fallback to standard parse if not multipart
		_ = r.ParseForm()
	}

	fields := []string{
		"company_name", "company_street", "company_city", "company_state",
		"company_zip", "company_country", "company_email", "company_phone", "company_currency",
		"default_terms",
	}

	for _, key := range fields {
		val := r.FormValue(key)
		h.saveSetting(ctx, tenant.ID, key, val)
	}

	// Handle company logo file upload
	file, header, err := r.FormFile("company_logo")
	if err == nil {
		defer file.Close()

		// Read all file bytes robustly using io.ReadAll
		buf, err := io.ReadAll(file)
		if err == nil && len(buf) > 0 {
			// Get mime type
			mimeType := header.Header.Get("Content-Type")
			if mimeType == "" {
				mimeType = "image/png" // default
			}

			// Encode to base64 data URL
			base64Str := base64.StdEncoding.EncodeToString(buf)
			dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)

			h.saveSetting(ctx, tenant.ID, "company_logo", dataURL)
		}
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
		SMTPBccEmail:  h.getSetting(ctx, tenant.ID, "smtp_bcc_email", ""),
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
		"smtp_host", "smtp_port", "smtp_user", "smtp_pass", "smtp_from_email", "smtp_from_name", "smtp_bcc_email",
	}

	for _, key := range fields {
		val := r.FormValue(key)
		h.saveSetting(ctx, tenant.ID, key, val)
	}

	SetFlash(w, "flash_success", "SMTP server configurations saved.")
	http.Redirect(w, r, "/settings/email", http.StatusSeeOther)
}

// TestEmail dispatches a real-time connection check message.
func (h *SettingsHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	smtpHost := r.FormValue("smtp_host")
	smtpPortStr := r.FormValue("smtp_port")
	smtpUser := r.FormValue("smtp_user")
	smtpPass := r.FormValue("smtp_pass")
	smtpFromEmail := r.FormValue("smtp_from_email")
	smtpFromName := r.FormValue("smtp_from_name")
	smtpBcc := r.FormValue("smtp_bcc_email")

	// Fallback to database configurations if form fields are blank
	if smtpHost == "" {
		smtpHost = h.getSetting(ctx, tenant.ID, "smtp_host", "")
	}
	if smtpPortStr == "" {
		smtpPortStr = h.getSetting(ctx, tenant.ID, "smtp_port", "")
	}
	if smtpUser == "" {
		smtpUser = h.getSetting(ctx, tenant.ID, "smtp_user", "")
	}
	if smtpPass == "" {
		smtpPass = h.getSetting(ctx, tenant.ID, "smtp_pass", "")
	}
	if smtpFromEmail == "" {
		smtpFromEmail = h.getSetting(ctx, tenant.ID, "smtp_from_email", "")
	}
	if smtpFromName == "" {
		smtpFromName = h.getSetting(ctx, tenant.ID, "smtp_from_name", "")
	}
	if smtpBcc == "" {
		smtpBcc = h.getSetting(ctx, tenant.ID, "smtp_bcc_email", "")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if smtpHost == "" || smtpPortStr == "" || smtpUser == "" || smtpPass == "" || smtpFromEmail == "" {
		w.Write([]byte(`<div class="li-flash li-flash--error" style="position:static; margin-bottom:var(--space-4); max-width:100%;">✗ Please fill in all required SMTP fields to send a test email.</div>`))
		return
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`<div class="li-flash li-flash--error" style="position:static; margin-bottom:var(--space-4); max-width:100%%;">✗ Invalid port number: %v</div>`, err)))
		return
	}

	m := mail.NewMailer(smtpHost, smtpPort, smtpUser, smtpPass, smtpFromEmail, smtpFromName, smtpBcc)
	
	subject := "LayerInvoice SMTP Connection Test"
	bodyHTML := "<h3>SMTP Mailer Connection Test Successful!</h3><p>Your LayerInvoice SMTP email server is correctly configured and successfully sending outbound mail.</p>"
	
	err = m.Send(smtpFromEmail, subject, bodyHTML)
	if err != nil {
		// Output structured logs and stdout details to standard debug console
		slog.Error("SMTP test connection failed", "error", err, "host", smtpHost, "port", smtpPort, "user", smtpUser)
		fmt.Printf("[Mailer Error] SMTP connection failed: %v\n", err)

		w.Write([]byte(fmt.Sprintf(`<div class="li-flash li-flash--error" style="position:static; margin-bottom:var(--space-4); max-width:100%%;">✗ SMTP Connection Failed: %v</div>`, err)))
		return
	}

	w.Write([]byte(`<div class="li-flash li-flash--success" style="position:static; margin-bottom:var(--space-4); max-width:100%;">✓ Connection Successful! Test email sent to ` + smtpFromEmail + `</div>`))
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
