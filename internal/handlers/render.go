package handlers

import (
	"embed"
	"html/template"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/money"
	"net/http"
	"path/filepath"
	"strings"
)

// PageData is the standard context passed to all page layouts.
type PageData struct {
	PageTitle     string
	ActiveMenu    string
	CurrentUser   *auth.User
	CurrentTenant *auth.Tenant
	FlashSuccess  string
	FlashError    string
	Data          interface{} // Page-specific context
}

// Render compiles and executes layout-based templates.
func Render(w http.ResponseWriter, r *http.Request, templatesFS embed.FS, layoutName string, templateName string, data interface{}, pageTitle string, activeMenu string) {
	// 1. Load active authenticated context
	user, _ := auth.GetUser(r.Context())
	tenant, _ := auth.GetTenant(r.Context())

	// 2. Fetch and consume flash messages
	flashSuccess := getFlash(w, r, "flash_success")
	flashError := getFlash(w, r, "flash_error")

	// 3. Set up template func map
	funcMap := template.FuncMap{
		"formatMoney": func(amount int64, currency string) string {
			return money.New(amount, currency).String()
		},
		"formatMoneyRaw": func(amount int64) string {
			return money.FormatAmount(amount)
		},
		"formatDate": func(dateStr string) string {
			// standard date formats or raw
			if len(dateStr) >= 10 {
				return dateStr[:10]
			}
			return dateStr
		},
		"hasRole": func(u *auth.User, role string) bool {
			if u == nil {
				return false
			}
			return u.Role == role
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"divFloat": func(a, b int64) float64 {
			return float64(a) / float64(b)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	}

	// 4. Resolve the layout path and template path
	layoutPath := filepath.Join("templates", "layout", layoutName+".html")
	templatePath := filepath.Join("templates", templateName)

	// In Go, paths in embed.FS must use forward slashes even on Windows!
	layoutPath = filepath.ToSlash(layoutPath)
	templatePath = filepath.ToSlash(templatePath)

	// 5. Parse templates dynamically
	tmpl, err := template.New(layoutName).Funcs(funcMap).ParseFS(templatesFS, layoutPath, templatePath)
	if err != nil {
		http.Error(w, "Template Parsing Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// For base layout, we execute "base". For auth layout, we execute "auth-layout".
	var targetTemplate string
	if layoutName == "base" {
		targetTemplate = "base"
	} else if layoutName == "auth" {
		targetTemplate = "auth-layout"
	} else {
		targetTemplate = layoutName
	}

	pageData := PageData{
		PageTitle:     pageTitle,
		ActiveMenu:    activeMenu,
		CurrentUser:   user,
		CurrentTenant: tenant,
		FlashSuccess:  flashSuccess,
		FlashError:    flashError,
		Data:          data,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, targetTemplate, pageData); err != nil {
		http.Error(w, "Template Execution Error: "+err.Error(), http.StatusInternalServerError)
	}
}

// RenderPartial executes page snippets without layout wrappers (perfect for HTMX updates).
func RenderPartial(w http.ResponseWriter, r *http.Request, templatesFS embed.FS, templatePath string, templateName string, data interface{}) {
	funcMap := template.FuncMap{
		"formatMoney": func(amount int64, currency string) string {
			return money.New(amount, currency).String()
		},
		"formatMoneyRaw": func(amount int64) string {
			return money.FormatAmount(amount)
		},
		"formatDate": func(dateStr string) string {
			if len(dateStr) >= 10 {
				return dateStr[:10]
			}
			return dateStr
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"divFloat": func(a, b int64) float64 {
			return float64(a) / float64(b)
		},
	}

	templatePath = filepath.ToSlash(templatePath)
	tmpl, err := template.New(templateName).Funcs(funcMap).ParseFS(templatesFS, templatePath)
	if err != nil {
		http.Error(w, "Partial Parsing Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Inject self-reference "Data" key if data is a map and doesn't have it,
	// to support templates that expect both .Data.Field and .Field contexts.
	if m, ok := data.(map[string]interface{}); ok {
		if _, exists := m["Data"]; !exists {
			m["Data"] = m
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	baseName := filepath.Base(templatePath)
	var execErr error
	if t := tmpl.Lookup(baseName); t != nil {
		execErr = tmpl.ExecuteTemplate(w, baseName, data)
	} else if t := tmpl.Lookup(templateName); t != nil {
		execErr = tmpl.ExecuteTemplate(w, templateName, data)
	} else {
		execErr = tmpl.Execute(w, data)
	}
	if execErr != nil {
		http.Error(w, "Partial Execution Error: "+execErr.Error(), http.StatusInternalServerError)
	}
}

// SetFlash is a helper to set a temporary session flash cookie.
func SetFlash(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   10, // short expiration
	})
}

func getFlash(w http.ResponseWriter, r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	// Consume cookie
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return cookie.Value
}
