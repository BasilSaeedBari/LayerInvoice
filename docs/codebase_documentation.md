# LayerInvoice — Go Codebase Reference Manual

This document provides a highly detailed, file-by-file codebase reference manual for **LayerInvoice**. It covers every module, structure, function signature, input parameter, return type, and operational purpose across the entire Go backend.

---

## 📂 Directory Structure Overview

LayerInvoice is organized as a clean, modular Go application using idiomatic structures:

```text
├── main.go                     # Application entrypoint & bootstraps migrations/background tasks
├── docs/                       # Architecture diagrams and design documents
├── internal/
│   ├── app/                    # Central application configuration context
│   ├── auth/                   # Password cryptography and SQLite session management
│   ├── calculator/             # Dynamic 3D print pricing formulas and arithmetic
│   ├── config/                 # Environment variables loader
│   ├── db/                     # SQLite database bootstrap and ULID generation
│   ├── handlers/               # HTTP Controller Actions, HTML Renderers, and routing table
│   ├── jobs/                   # Periodic background schedulers
│   ├── mail/                   # SMTP client for background email dispatches
│   ├── money/                  # Parsing and formatting calculations in Paisa units
│   └── pdf/                    # Programmatic PDF generation engine
└── templates/                  # HTML components and HTMX layouts
```

---

## 1. ⚙️ Core Bootstrapping & Configuration

### 📄 `main.go`
The entrypoint of the application. It loads configurations, opens database connections, triggers Goose migrations, spins up background cron jobs, and starts the HTTP server.

- **`main()`**
  - **Inputs**: None (retrieves command-line flags and environment variables).
  - **Outputs**: None.
  - **Description**: Bootstraps the application. In development environment (`cfg.AppEnv == "development"`), it automatically purges `layerinvoice.db`, `layerinvoice.db-wal`, and `layerinvoice.db-shm` files for a clean database reload.

---

### 📄 `internal/app/app.go`
Manages the central dependency injection struct representing the global application state.

- **`type App struct`**
  - **Fields**:
    - `Config *config.Config`: Application configuration parameters.
    - `DB *sql.DB`: SQLite database connection.
    - `SessionStore *auth.SessionStore`: Handler for user login states.
    - `TemplatesFS embed.FS`: Embedded virtual filesystem containing HTML templates.
  - **Purpose**: Passed to handlers to prevent circular dependency cycles.

---

### 📄 `internal/config/config.go`
Loads settings from the filesystem or environment.

- **`type Config struct`**
  - **Fields**: `Port`, `AppEnv`, `DbPath`, `SessionSecret`, `SmtpHost`, `SmtpPort`, `SmtpUser`, `SmtpPass`, `SmtpFrom`.
- **`LoadConfig() (*Config, error)`**
  - **Inputs**: None (reads `.env` or system environment).
  - **Outputs**: `(*Config, error)`
  - **Description**: Compiles all configurations into a single struct. Sets default values for the database path (`layerinvoice.db`) and port (`8080`).

---

### 📄 `internal/db/db.go`
Initializes the SQLite database driver and generates unique identifiers.

- **`Connect(cfg *config.Config) (*sql.DB, error)`**
  - **Inputs**: `cfg *config.Config`
  - **Outputs**: `(*sql.DB, error)`
  - **Description**: Configures and opens a CGO-free, pure-Go SQLite connection. Automatically enables Write-Ahead Logging (`PRAGMA journal_mode=WAL`), foreign keys (`PRAGMA foreign_keys=ON`), and database busy-timeouts.
- **`NewULID() string`**
  - **Inputs**: None.
  - **Outputs**: `string`
  - **Description**: Generates a lexicographically sortable, universally unique ULID string.

---

## 2. 🔐 Security, Cryptography & Sessions

### 📄 `internal/auth/password.go`
Provides high-performance password hashing wrappers using standard bcrypt algorithm.

- **`HashPassword(password string) (string, error)`**
  - **Inputs**: `password string` (plaintext password).
  - **Outputs**: `(string, error)` (hashed password).
  - **Description**: Computes a secure bcrypt hash of a plaintext password.
- **`CheckPasswordHash(password, hash string) bool`**
  - **Inputs**: `password string`, `hash string`
  - **Outputs**: `bool` (true if matching).
  - **Description**: Compares a plaintext password against a pre-computed hash.

---

### 📄 `internal/auth/session.go`
Manages browser session validation stored securely inside the SQLite `sessions` database table.

- **`NewSessionStore(db *sql.DB) *SessionStore`**
  - **Inputs**: `db *sql.DB`
  - **Outputs**: `*SessionStore`
- **`(s *SessionStore) CreateSession(userID string, maxAge time.Duration) (string, error)`**
  - **Inputs**: `userID string`, `maxAge time.Duration`
  - **Outputs**: `(string, error)` (returns session token / ID).
  - **Description**: Creates a session record inside SQLite and returns a ULID token representing it.
- **`(s *SessionStore) GetSession(sessionID string) (*User, error)`**
  - **Inputs**: `sessionID string`
  - **Outputs**: `(*User, error)` (User containing ID, Name, Email, TenantID).
  - **Description**: Retrieves and validates session expiry status, returning the user info.
- **`(s *SessionStore) DeleteSession(sessionID string) error`**
  - **Inputs**: `sessionID string`
  - **Outputs**: `error`
  - **Description**: Deletes a session from the database (logout behavior).

---

### 📄 `internal/auth/middleware.go`
HTTP middleware mapping routes to protected user/tenant contexts.

- **`SessionMiddleware(store *SessionStore) func(http.Handler) http.Handler`**
  - **Inputs**: `store *SessionStore`
  - **Outputs**: `func(http.Handler) http.Handler`
  - **Description**: Extracts session cookies and injects the active User/Tenant structs directly into the HTTP Request Context.
- **`RequireAuth(next http.Handler) http.Handler`**
  - **Inputs**: `next http.Handler`
  - **Outputs**: `http.Handler`
  - **Description**: Rejects requests and redirects unauthorized requests to `/login`.
- **`GetTenant(ctx context.Context) (*Tenant, error)`**
  - **Inputs**: `ctx context.Context`
  - **Outputs**: `(*Tenant, error)`
  - **Description**: Helper to retrieve the active tenant context safely.
- **`GetUser(ctx context.Context) (*User, error)`**
  - **Inputs**: `ctx context.Context`
  - **Outputs**: `(*User, error)`
  - **Description**: Helper to retrieve the active logged-in user struct safely.

---

## 3. 📐 Advanced Math & Core Engines

### 📄 `internal/calculator/print_cost.go`
Calculates comprehensive, multi-variable 3D printing cost breakdowns.

- **`type PrintJobInputs struct`**
  - Inputs for material weight, print hours, markups, machine wear, labor, failure risk percentage, shipping, and profit buffers.
- **`type PrintJobBreakdown struct`**
  - Detailed outputs: raw filament costs, filament profit markup, machine wear, failure buffers, total production cost, final profit markup, and estimated final price.
- **`Calculate(in PrintJobInputs) PrintJobBreakdown`**
  - **Inputs**: `in PrintJobInputs`
  - **Outputs**: `PrintJobBreakdown`
  - **Description**: Processes all parameters using standard rounding logic. Important: Profit Markups and Risk Allowances are evaluated using direct percentage values (e.g. `100` = 100% markup) rather than basis points.

---

### 📄 `internal/money/money.go`
Ensures error-free currency storage by parsing decimal values and converting them to internal integer Paisa units (smallest currency unit, e.g. `Rs 1.00 = 100 Paisa`).

- **`type Money struct`**
  - **Fields**: `Amount int64`, `Currency string`.
- **`Parse(valStr, currency string) (Money, error)`**
  - **Inputs**: `valStr string` (e.g. `"150.50"`), `currency string` (e.g. `"PKR"`).
  - **Outputs**: `(Money, error)`
  - **Description**: Converts standard float-based text prices to integer Paisa units safely.
- **`FormatAmount(amount int64) string`**
  - **Inputs**: `amount int64`
  - **Outputs**: `string` (e.g. `"150.50"`)
  - **Description**: Formats an integer amount back into a decimal text.
- **`FormatMoney(amount int64, currency string) string`**
  - **Inputs**: `amount int64`, `currency string`
  - **Outputs**: `string` (e.g. `"Rs 150.50"`)
  - **Description**: Prepends currency symbols dynamically depending on the selected currency code.

---

### 📄 `internal/pdf/pdf.go`
Programmatic A4 PDF generator using the lightweight Go-PDF library.

- **`Generate(doc PDFDocument) ([]byte, error)`**
  - **Inputs**: `doc PDFDocument` (Seller/Buyer info, currency, line items, totals, notes).
  - **Outputs**: `([]byte, error)` (returns a raw PDF byte array).
  - **Description**: Programmatically compiles standard billing layouts, addresses, detailed item grids, financial totals, notes, and footers in A4 page size.

---

### 📄 `internal/mail/mailer.go`
Handles asynchronous SMTP dispatch for invoices and quotes.

- **`SendEmail(smtpHost string, smtpPort int, smtpUser, smtpPass, smtpFrom, to, subject, body string, attachment []byte, attachmentName string) error`**
  - **Inputs**: SMTP credentials, receiver email, subject, body, attachment bytes, and filename.
  - **Outputs**: `error`
  - **Description**: Opens an SMTP TLS/Plain connection, compiles standard multipart emails, appends the PDF, and sends it.

---

### 📄 `internal/jobs/scheduler.go`
Handles periodic background routines.

- **`StartScheduler(db *sql.DB)`**
  - **Inputs**: `db *sql.DB`
  - **Outputs**: None.
  - **Description**: Spins up a background ticker (every 24 hours) to run cron tasks like automatically identifying and flagging overdue invoices.

---

## 4. 🔗 Route Management & Handlers

### 📄 `internal/handlers/router.go`
Wires chi controllers to protected URL paths.

- **`WireRoutes(a *app.App, staticFS embed.FS) *chi.Mux`**
  - **Inputs**: `a *app.App`, `staticFS embed.FS`
  - **Outputs**: `*chi.Mux`
  - **Description**: Orchestrates middleware setups, exposes virtual files, configures public auth gateways, and handles protected multi-tenant modules.

---

### 📄 `internal/handlers/render.go`
Centralizes server-side template assembly.

- **`Render(w http.ResponseWriter, r *http.Request, fs embed.FS, layout, templateName string, data interface{}, pageTitle, activeNav string)`**
  - **Inputs**: ResponseWriter, Request, filesystem, layouts, templates, context data, title, and CSS active flags.
  - **Outputs**: None (writes HTML response).
  - **Description**: Compiles base layers and main templates to deliver standard unified layouts.
- **`RenderPartial(w http.ResponseWriter, r *http.Request, fs embed.FS, partialPath, blockName string, data interface{})`**
  - **Inputs**: ResponseWriter, Request, filesystem, template path, block identifier, and data context.
  - **Outputs**: None.
  - **Description**: Renders and returns a specific, small HTML block for HTMX replacements.
- **`SetFlash(w http.ResponseWriter, name, value string)`**
  - **Inputs**: ResponseWriter, Cookie key name, Cookie value text.
  - **Outputs**: None.
  - **Description**: Sets temporary HTTP cookies for rendering screen notifications.

---

### 📄 `internal/handlers/auth.go`
Exposes login/logout credentials processing.

- **`ShowLogin(w http.ResponseWriter, r *http.Request)`**: Renders login page.
- **`HandleLogin(w http.ResponseWriter, r *http.Request)`**: Validates passwords and drops authorization session cookies.
- **`HandleLogout(w http.ResponseWriter, r *http.Request)`**: Clears sessions and redirects to the login screen.

---

### 📄 `internal/handlers/dashboard.go`
Summarizes financial metrics and fabrication totals.

- **`Index(w http.ResponseWriter, r *http.Request)`**
  - **Inputs**: ResponseWriter, Request.
  - **Outputs**: None.
  - **Description**: Queries aggregate stats for revenue, unpaid invoice trackers, and pending quotes. Computes **Print Hours Billed** and **Filament weight billed** this month exclusively from invoices marked as `paid` or `partially_paid` to ensure metrics correspond with active paid volume.

---

### 📄 `internal/handlers/clients.go`
Manages CRM Client accounts.

- **`List(...)`**: Displays the active list of clients.
- **`ShowNewForm(...)`**: Renders blank registration forms (passes empty context maps to prevent nil template pointer crashes).
- **`Create(...)`**: Inserts client records.
- **`Show(...)`**: Displays a detailed client card, active invoices, total billed aggregates, and payment ratios.
- **`ShowEditForm(...)`**: Loads form fields with existing records (correctly parses SQLite `is_active` integers into Go booleans).
- **`Update(...)`**: Saves client profile edits.
- **`Delete(...)`**: Soft-deletes a client profile by setting `is_active = 0` to preserve historical document consistency.

---

### 📄 `internal/handlers/filaments.go`
Preserves custom material reels presets.

- **`List(...)`**: Lists registered material presets.
- **`ShowNewForm(...)`**: Renders new filament registration cards.
- **`Create(...)`**: Computes cost-per-gram properties automatically and inserts records.
- **`Delete(...)`**: Removes filament presets.
- **`GetCostPerGram(...)`**: Inline HTMX action that returns raw unit price text for calculations.

---

### 📄 `internal/handlers/quotes.go`
Controls client estimation flow.

- **`List(...)`**: Queries and lists quotes (correctly resolves client `contact_name` fields via SQL joins).
- **`Show(...)`**: Loads the interactive quote builder.
- **`AddPrintItem(w http.ResponseWriter, r *http.Request)`**
  - **Inputs**: Forms containing 3D print specifications.
  - **Outputs**: Renders an inline row partial dynamically.
  - **Description**: Evaluates materials, print times, and overhead fees. Saves parameters and **automatically generates an exhaustive, multi-line sub-description** listing filament brand, material type, grams used, print hours, machine wear, and extra overheads, ensuring professional quote details.
- **`ConvertToInvoice(w http.ResponseWriter, r *http.Request)`**
  - **Inputs**: Quote ULID.
  - **Outputs**: Redirects to the generated invoice.
  - **Description**: Atomic, transaction-safe 1-click quote conversion. Clones the quote record into an invoice and **duplicates all line items perfectly, maintaining their exact `item_type` properties (preserving `3d_print` types)** so volume calculations compile correctly.
- **`Delete(...)`**: Transactionally purges the quote and its child line items.

---

### 📄 `internal/handlers/invoices.go`
Controls client billing lifecycle, updates, live cost calculations, and email workers.

- **`List(...)`**: Lists tenant invoices.
- **`Show(...)`**: Loads the interactive invoice builder.
- **`AddPrintItem(...)`**: Evaluates 3D print specifications, inserts the line item, and **automatically generates the dynamic multi-line sub-description** (detailing filament company/brand, grams used, hours, and overhead breakdown) for clean screen rendering and PDF creation.
- **`LivePreview(w http.ResponseWriter, r *http.Request)`**
  - **Inputs**: Live modal calculator parameters.
  - **Outputs**: Renders a dynamic calculation breakdown HTML block.
  - **Description**: Computes print costs live on every keypress or selection. Markup and failure variables are processed directly using standard percentages.
- **`TransitionStatus(...)`**: Updates status. When changing to `'sent'`, it executes an immediate background email dispatch.
- **`Delete(...)`**: Stops debounced email background timers, starts transaction, and deletes the invoice (SQLite cascades deletion to payment registries, causing all financial totals to update dynamically).

---

### 📄 `internal/handlers/payments.go`
Registers payment receipts against invoices.

- **`List(...)`**: Displays registered payment logs.
- **`ShowNewModal(...)`**: Renders the payment logger popup with remaining balances.
- **`Create(w http.ResponseWriter, r *http.Request)`**
  - **Inputs**: Invoice ID, amount, payment method, reference, and date.
  - **Outputs**: Redirects back to the invoice page.
  - **Description**: Transactionally inserts a payment receipt and updates invoice metrics (`amount_paid` and `balance_due`). If the balance is zero, status is automatically transitioned to `'paid'`, instantly updating outstanding statistics and 3D printing dashboard totals!

---

### 📄 `internal/handlers/settings.go`
Updates company details and SMTP mail configurations.

- **`ShowCompany(...)` / `SaveCompany(...)`**: Saves company details. When updating company name, it automatically runs an SQL update on the active organizational `tenants` record to instantly refresh the brand header navigation badge.
- **`ShowEmail(...)` / `SaveEmail(...)`**: Saves email SMTP credentials.
- **`TestEmail(...)`**: Outbound SMTP validator. Triggered via HTMX directly from the SMTP configuration screen. Performs real-time server connections, logs connection events and errors to both stdout and structured slog logs, and outputs inline checkmark or diagnostic error panels.

---

### 📄 `internal/handlers/system.go`
Exposes core setup configurations.

- **`ShowSetup(...)` / `HandleSetup(...)`**: Setup bootstrapper shown on clean installations.
- **`ShowSystem(...)` / `CreateUser(...)`**: Renders administrative screens.

