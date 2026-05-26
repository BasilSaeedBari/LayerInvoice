package handlers

import (
	"layerinvoice/internal/app"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/calculator"
	"layerinvoice/internal/db"
	"layerinvoice/internal/money"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// FilamentsHandler manages 3D print filament profiles.
type FilamentsHandler struct {
	app *app.App
}

// NewFilamentsHandler creates a new FilamentsHandler.
func NewFilamentsHandler(a *app.App) *FilamentsHandler {
	return &FilamentsHandler{app: a}
}

// FilamentViewModel structures variables for visual filament profile listings.
type FilamentViewModel struct {
	ID           string
	Name         string
	Material     string
	Color        string
	Brand        string
	SpoolCost    int64
	SpoolWeightG int
	CostPerGram  int64
	Currency     string
	Notes        string
	IsActive     bool
}

// List queries and displays all active filament presets.
func (h *FilamentsHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	query := `
		SELECT id, name, material, COALESCE(color, ''), COALESCE(brand, ''), 
		       spool_cost, spool_weight_g, cost_per_gram, currency, COALESCE(notes, ''), is_active
		FROM filament_profiles
		WHERE tenant_id = ? AND is_active = 1
		ORDER BY name ASC
	`

	rows, err := h.app.DB.QueryContext(r.Context(), query, tenant.ID)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var filaments []FilamentViewModel
	for rows.Next() {
		var f FilamentViewModel
		var activeInt int
		err := rows.Scan(
			&f.ID, &f.Name, &f.Material, &f.Color, &f.Brand,
			&f.SpoolCost, &f.SpoolWeightG, &f.CostPerGram, &f.Currency, &f.Notes, &activeInt,
		)
		if err == nil {
			f.IsActive = activeInt == 1
			filaments = append(filaments, f)
		}
	}

	Render(w, r, h.app.TemplatesFS, "base", "filaments/list.html", filaments, "Filament Profiles", "filaments")
}

// ShowNewForm renders the Filament preset creation view.
func (h *FilamentsHandler) ShowNewForm(w http.ResponseWriter, r *http.Request) {
	Render(w, r, h.app.TemplatesFS, "base", "filaments/form.html", nil, "New Filament", "filaments")
}

// Create stores a new Filament preset, auto-calculating cost-per-gram details in raw cents/paisa.
func (h *FilamentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	material := r.FormValue("material")
	spoolCostStr := r.FormValue("spool_cost")
	spoolWeightStr := r.FormValue("spool_weight_g")
	currency := r.FormValue("currency")

	if name == "" || material == "" || spoolCostStr == "" || spoolWeightStr == "" {
		SetFlash(w, "flash_error", "Name, material, spool cost, and spool weight are required.")
		http.Redirect(w, r, "/filaments/new", http.StatusSeeOther)
		return
	}

	// Parse spool cost to Money
	spoolCostMoney, err := money.Parse(spoolCostStr, currency)
	if err != nil {
		SetFlash(w, "flash_error", "Invalid spool cost amount: "+err.Error())
		http.Redirect(w, r, "/filaments/new", http.StatusSeeOther)
		return
	}

	// Parse weight
	spoolWeightG, err := strconv.Atoi(spoolWeightStr)
	if err != nil || spoolWeightG <= 0 {
		SetFlash(w, "flash_error", "Invalid spool weight (must be positive integer).")
		http.Redirect(w, r, "/filaments/new", http.StatusSeeOther)
		return
	}

	// Calculate cost per gram precisely (ROUND_HALF_UP)
	costPerGram := calculator.RoundHalfUp(float64(spoolCostMoney.Amount) / float64(spoolWeightG))

	id := db.NewULID()
	now := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO filament_profiles (
			id, tenant_id, name, material, color, brand, spool_cost, spool_weight_g, 
			cost_per_gram, currency, notes, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`

	_, err = h.app.DB.ExecContext(r.Context(), query,
		id, tenant.ID, name, material,
		r.FormValue("color"),
		r.FormValue("brand"),
		spoolCostMoney.Amount,
		spoolWeightG,
		costPerGram,
		currency,
		r.FormValue("notes"),
		now, now,
	)

	if err != nil {
		SetFlash(w, "flash_error", "Database error: "+err.Error())
		http.Redirect(w, r, "/filaments/new", http.StatusSeeOther)
		return
	}

	SetFlash(w, "flash_success", "Filament profile added successfully.")
	http.Redirect(w, r, "/filaments", http.StatusSeeOther)
}

// Delete marks a Filament preset as inactive (soft delete).
func (h *FilamentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	id := chi.URLParam(r, "id")

	query := `UPDATE filament_profiles SET is_active = 0, updated_at = ? WHERE tenant_id = ? AND id = ?`
	_, err := h.app.DB.ExecContext(r.Context(), query, time.Now().Format(time.RFC3339), tenant.ID, id)

	if err != nil {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Trigger", `{"flash_error": "Failed to delete profile"}`)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		SetFlash(w, "flash_error", "Failed to delete profile.")
		http.Redirect(w, r, "/filaments", http.StatusSeeOther)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", `{"flash_success": "Filament profile deleted"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	SetFlash(w, "flash_success", "Filament profile deleted.")
	http.Redirect(w, r, "/filaments", http.StatusSeeOther)
}

// GetCostPerGram handles inline HTMX fetches returning raw values.
func (h *FilamentsHandler) GetCostPerGram(w http.ResponseWriter, r *http.Request) {
	tenant, _ := auth.GetTenant(r.Context())
	id := r.FormValue("filament_id")

	var cpg int64
	var currency string
	err := h.app.DB.QueryRowContext(r.Context(), `
		SELECT cost_per_gram, currency FROM filament_profiles WHERE tenant_id = ? AND id = ? AND is_active = 1
	`, tenant.ID, id).Scan(&cpg, &currency)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
		return
	}

	formatted := money.FormatAmount(cpg)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(formatted))
}
