package handlers

import (
	"embed"
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// WireRoutes maps all URL routes to their respective controllers and setups middlewares.
func WireRoutes(a *app.App, staticFS embed.FS) *chi.Mux {
	r := chi.NewRouter()

	// 1. Core Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Session context injection
	r.Use(auth.SessionMiddleware(a.SessionStore))

	// 2. Static Assets Routing (embedded)
	r.Handle("/static/*", http.FileServer(http.FS(staticFS)))

	// 3. Controller Instances
	authH := NewAuthHandler(a)
	dashH := NewDashboardHandler(a)
	clientH := NewClientsHandler(a)
	filH := NewFilamentsHandler(a)
	quoteH := NewQuotesHandler(a)
	invH := NewInvoicesHandler(a)
	payH := NewPaymentsHandler(a)
	setH := NewSettingsHandler(a)
	sysH := NewSystemHandler(a)

	// 4. Setup Bootstrapper (Public if database is empty)
	r.Get("/setup", sysH.ShowSetup)
	r.Post("/setup", sysH.HandleSetup)

	// 5. Authentication Routes
	r.Get("/login", authH.ShowLogin)
	r.Post("/login", authH.HandleLogin)
	r.Post("/logout", authH.HandleLogout)

	// 6. Live Cost Calculations (HTMX updates)
	r.Post("/calc/print-cost", invH.LivePreview)

	// 7. Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)

		// Dashboard
		r.Get("/", dashH.Index)

		// Clients CRM
		r.Route("/clients", func(r chi.Router) {
			r.Get("/", clientH.List)
			r.Get("/new", clientH.ShowNewForm)
			r.Post("/", clientH.Create)
			r.Get("/{id}", clientH.Show)
			r.Get("/{id}/edit", clientH.ShowEditForm)
			r.Post("/{id}", clientH.Update)
			r.Post("/{id}/delete", clientH.Delete)
		})

		// Filament Profiles Presets
		r.Route("/filaments", func(r chi.Router) {
			r.Get("/", filH.List)
			r.Get("/new", filH.ShowNewForm)
			r.Post("/", filH.Create)
			r.Delete("/{id}", filH.Delete)
			r.Post("/cost-per-gram", filH.GetCostPerGram) // inline HTMX select lookup
		})

		// Estimates & Quotes
		r.Route("/quotes", func(r chi.Router) {
			r.Get("/", quoteH.List)
			r.Get("/new", quoteH.ShowNewForm)
			r.Post("/", quoteH.Create)
			r.Get("/{id}", quoteH.Show)
			
			// Dynamic Quote Line Items (HTMX)
			r.Get("/{id}/line-items/new-custom", quoteH.AddCustomRow)
			r.Post("/{id}/line-items/{li_id}/update", quoteH.UpdateRow)
			r.Delete("/{id}/line-items/{li_id}", quoteH.DeleteRow)
			r.Get("/{id}/totals", quoteH.Totals)
			r.Post("/{id}/discount-tax", quoteH.UpdateDiscountTax)
			r.Post("/{id}/line-items/3d-print-add", quoteH.AddPrintItem)
			
			// Transitions & Conversions
			r.Get("/{id}/status", quoteH.TransitionStatus)
			r.Post("/{id}/convert", quoteH.ConvertToInvoice)
			r.Get("/{id}/pdf", quoteH.ShowPDF)
			r.Post("/{id}/delete", quoteH.Delete)
		})

		// Invoices & Billing
		r.Route("/invoices", func(r chi.Router) {
			r.Get("/", invH.List)
			r.Get("/new", invH.ShowNewForm)
			r.Post("/", invH.Create)
			r.Get("/{id}", invH.Show)
			
			// Dynamic Invoice Line Items (HTMX)
			r.Get("/{id}/line-items/new-custom", invH.AddCustomRow)
			r.Post("/{id}/line-items/{li_id}/update", invH.UpdateRow)
			r.Delete("/{id}/line-items/{li_id}", invH.DeleteRow)
			r.Get("/{id}/totals", invH.Totals)
			r.Post("/{id}/discount-tax", invH.UpdateDiscountTax)
			r.Post("/{id}/line-items/3d-print-add", invH.AddPrintItem)
			
			// Transitions
			r.Get("/{id}/status", invH.TransitionStatus)
			r.Get("/{id}/pdf", invH.ShowPDF)
			r.Post("/{id}/delete", invH.Delete)
		})

		// Payments Register
		r.Route("/payments", func(r chi.Router) {
			r.Get("/", payH.List)
			r.Get("/new", payH.ShowNewModal)
			r.Post("/", payH.Create)
		})
		r.Post("/payment-methods", payH.AddMethod)

		// Settings & System configuration
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", setH.ShowCompany)
			r.Post("/", setH.SaveCompany)
			r.Get("/email", setH.ShowEmail)
			r.Post("/email", setH.SaveEmail)
			r.Post("/email/test", setH.TestEmail)
		})

		r.Route("/system", func(r chi.Router) {
			r.Get("/", sysH.ShowSystem)
			r.Post("/users", sysH.CreateUser)
			r.Post("/tax-rates", sysH.CreateTaxRate)
		})
	})

	return r
}
