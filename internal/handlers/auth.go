package handlers

import (
	"database/sql"
	"errors"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"net/http"
	"time"
)

// AuthHandler handles authentication-related HTTP routes.
type AuthHandler struct {
	app *app.App
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(a *app.App) *AuthHandler {
	return &AuthHandler{app: a}
}

// ShowLogin renders the login page.
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if user, _ := auth.GetUser(r.Context()); user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	Render(w, r, h.app.TemplatesFS, "auth", "auth/login.html", nil, "Login", "")
}

// HandleLogin processes credentials and establishes secure sessions.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		SetFlash(w, "flash_error", "Email and password are required.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var userID, tenantID, hash string
	var isActive int

	query := `SELECT id, tenant_id, password_hash, is_active FROM users WHERE email = ? LIMIT 1`
	err := h.app.DB.QueryRowContext(r.Context(), query, email).Scan(&userID, &tenantID, &hash, &isActive)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SetFlash(w, "flash_error", "Invalid email or password.")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		SetFlash(w, "flash_error", "Database query error.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if isActive != 1 {
		SetFlash(w, "flash_error", "Account has been deactivated.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Verify hashed password
	if !auth.CheckPasswordHash(password, hash) {
		SetFlash(w, "flash_error", "Invalid email or password.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Create new session valid for 24 hours
	sessionDuration := 24 * time.Hour
	sessionID, err := h.app.SessionStore.CreateSession(r.Context(), userID, sessionDuration)
	if err != nil {
		SetFlash(w, "flash_error", "Failed to initialize session.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "layerinvoice_session",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(sessionDuration),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	SetFlash(w, "flash_success", "Welcome back!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleLogout clears session token and redirects to login page.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("layerinvoice_session")
	if err == nil {
		// Revoke in DB
		_ = h.app.SessionStore.DeleteSession(r.Context(), cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "layerinvoice_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	SetFlash(w, "flash_success", "You have been logged out.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
