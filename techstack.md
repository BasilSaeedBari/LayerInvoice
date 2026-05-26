# LayerInvoice — Tech Stack

## Summary Table

| Layer              | Technology                        | Notes                                          |
| ------------------ | --------------------------------- | ---------------------------------------------- |
| Language           | Go 1.26+                          | Standard library first; minimal dependencies   |
| HTTP Router        | `go-chi/chi` v5                   | Tiny, idiomatic, middleware-friendly           |
| Templates          | `html/template` (stdlib)          | No external template engine; no hydration      |
| Reactive UI        | HTMX 2.x                          | Partial HTML swaps; no full-page reloads       |
| JS Enhancement     | Alpine.js v3 (optional/minimal)   | Dropdowns, toggles only; no state machines     |
| CSS Framework      | PicoCSS v2                        | Classless semantic HTML; custom vars on top    |
| Custom CSS         | Plain CSS variables               | Layered over PicoCSS; design tokens file       |
| Database           | SQLite (via `modernc.org/sqlite`) | Pure Go driver; no CGo required                |
| Query Layer        | Raw SQL + `database/sql`          | No ORM; hand-written queries in `/internal/db` |
| Migrations         | `pressly/goose` v3                | SQL migration files in `/migrations`           |
| PDF Generation     | `go-pdf/fpdf` v2                  | Pure Go; no headless browser required          |
| Email              | `gopkg.in/gomail.v2`              | SMTP only; configurable per-tenant             |
| Auth               | Custom session auth (bcrypt)      | Sessions in SQLite; no JWT, no OAuth           |
| Password Hashing   | `golang.org/x/crypto/bcrypt`      | Cost factor 12                                 |
| IDs                | ULID (`oklog/ulid` v2)            | Sortable, URL-safe, non-enumerable             |
| Money Precision    | Integer arithmetic (int64)        | Store smallest unit (paisa/cents); display via helper |
| Background Jobs    | `go-co-op/gocron` v2              | Recurring invoices; overdue checks             |
| Config             | Environment variables + `.env`    | `joho/godotenv` for local dev                  |
| Secrets            | Docker secrets or env vars        | No plaintext credentials in source             |
| Containerisation   | Docker + Docker Compose           | Single container; mounted SQLite volume        |
| Reverse Proxy      | Caddy v2                          | Automatic TLS; minimal config                  |
| Build              | `go build` only                   | No webpack, no vite, no npm, no node           |
| Asset Serving      | `net/http` static file server     | Embedded via `embed.FS`                        |
| Realtime           | SSE (Server-Sent Events)          | Dashboard updates; invoice status changes      |
| Logging            | `log/slog` (stdlib, Go 1.21+)     | Structured JSON logs                           |
| Testing            | `testing` stdlib + `testify`      | Table-driven tests; no heavy test framework    |

---

## Project Structure

```
layerinvoice/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point; wire everything together
├── internal/
│   ├── app/
│   │   └── app.go                   # App struct; holds DB, config, mailer
│   ├── auth/
│   │   ├── session.go               # Session creation, validation, deletion
│   │   ├── middleware.go            # RequireAuth, RequireRole middleware
│   │   └── password.go              # bcrypt helpers
│   ├── config/
│   │   └── config.go                # Load env vars into Config struct
│   ├── db/
│   │   ├── db.go                    # Open SQLite; run migrations
│   │   ├── queries/                 # One .sql file per domain entity
│   │   │   ├── clients.sql
│   │   │   ├── invoices.sql
│   │   │   ├── quotes.sql
│   │   │   ├── payments.sql
│   │   │   ├── filaments.sql
│   │   │   ├── line_items.sql
│   │   │   ├── users.sql
│   │   │   ├── settings.sql
│   │   │   └── audit.sql
│   │   └── models/                  # Go structs mirroring DB rows
│   │       ├── client.go
│   │       ├── invoice.go
│   │       ├── quote.go
│   │       ├── payment.go
│   │       ├── filament.go
│   │       └── ...
│   ├── handlers/                    # One file per top-level route group
│   │   ├── dashboard.go
│   │   ├── invoices.go
│   │   ├── quotes.go
│   │   ├── clients.go
│   │   ├── payments.go
│   │   ├── filaments.go
│   │   ├── settings.go
│   │   ├── system.go
│   │   ├── auth.go
│   │   └── sse.go
│   ├── money/
│   │   └── money.go                 # Integer money type; format/parse helpers
│   ├── pdf/
│   │   └── pdf.go                   # Invoice + quote PDF generation
│   ├── mail/
│   │   └── mailer.go                # Send mail via gomail; template rendering
│   ├── jobs/
│   │   └── scheduler.go             # gocron setup; recurring invoice job
│   ├── calculator/
│   │   └── print_cost.go            # 3D print cost engine (pure functions)
│   └── statemachine/
│       └── invoice.go               # Invoice/quote state transition logic
├── migrations/
│   ├── 00001_init_schema.sql
│   ├── 00002_add_filaments.sql
│   └── ...
├── templates/
│   ├── layout/
│   │   ├── base.html                # Root layout; nav, sidebar, flash
│   │   ├── auth.html                # Auth-only layout (login page)
│   │   └── partials/
│   │       ├── nav.html
│   │       ├── sidebar.html
│   │       ├── flash.html
│   │       ├── pagination.html
│   │       └── modal.html
│   ├── dashboard/
│   │   └── index.html
│   ├── invoices/
│   │   ├── list.html
│   │   ├── show.html
│   │   ├── form.html                # Create + Edit (same template)
│   │   └── partials/
│   │       ├── line_item_row.html   # HTMX partial for adding rows
│   │       ├── totals.html          # HTMX partial for live total update
│   │       └── status_badge.html
│   ├── quotes/
│   │   ├── list.html
│   │   ├── show.html
│   │   ├── form.html
│   │   └── partials/
│   │       └── line_item_row.html
│   ├── clients/
│   │   ├── list.html
│   │   ├── show.html                # Full client ledger view
│   │   └── form.html
│   ├── payments/
│   │   ├── list.html
│   │   └── partials/
│   │       └── method_row.html
│   ├── filaments/
│   │   ├── list.html
│   │   └── form.html
│   ├── settings/
│   │   ├── company.html
│   │   ├── email.html
│   │   ├── invoice_settings.html
│   │   └── system.html
│   ├── auth/
│   │   └── login.html
│   └── pdf/                         # PDF-only HTML templates (rendered to PDF)
│       ├── invoice.html
│       └── quote.html
├── static/
│   ├── css/
│   │   ├── pico.min.css             # PicoCSS v2 (vendored)
│   │   └── app.css                  # Custom tokens + overrides
│   ├── js/
│   │   ├── htmx.min.js              # HTMX 2.x (vendored)
│   │   └── alpine.min.js            # Alpine.js v3 (vendored)
│   └── img/
│       └── placeholder-logo.svg
├── docker/
│   ├── Dockerfile
│   └── Caddyfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

---

## Database Schema — SQLite

All monetary values stored as `INTEGER` (smallest unit, e.g., paisa for PKR, cents for USD).
All IDs are `TEXT` (ULID format).
All timestamps are `TEXT` (RFC3339 / ISO 8601).

### Core Tables

```sql
-- tenants (multi-company support)
CREATE TABLE tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- users
CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL REFERENCES tenants(id),
    name            TEXT NOT NULL,
    email           TEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'employee', -- owner | employee | viewer
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE(tenant_id, email)
);

-- sessions
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

-- settings (key-value per tenant)
CREATE TABLE settings (
    tenant_id   TEXT NOT NULL REFERENCES tenants(id),
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    PRIMARY KEY (tenant_id, key)
);

-- clients
CREATE TABLE clients (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id),
    company_name        TEXT,
    contact_name        TEXT NOT NULL,
    email               TEXT,
    phone               TEXT,
    whatsapp            TEXT,
    website             TEXT,
    tax_id              TEXT,
    billing_street      TEXT,
    billing_city        TEXT,
    billing_state       TEXT,
    billing_zip         TEXT,
    billing_country     TEXT,
    shipping_street     TEXT,
    shipping_city       TEXT,
    shipping_state      TEXT,
    shipping_zip        TEXT,
    shipping_country    TEXT,
    preferred_currency  TEXT NOT NULL DEFAULT 'PKR',
    payment_terms_days  INTEGER NOT NULL DEFAULT 30,
    default_tax_rate    INTEGER NOT NULL DEFAULT 0,  -- stored as basis points (1% = 100)
    default_discount    INTEGER NOT NULL DEFAULT 0,  -- stored as basis points
    notes               TEXT,
    tags                TEXT,  -- JSON array
    category            TEXT,
    is_active           INTEGER NOT NULL DEFAULT 1,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);

-- filament_profiles
CREATE TABLE filament_profiles (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL REFERENCES tenants(id),
    name            TEXT NOT NULL,
    material        TEXT NOT NULL,  -- PLA | PETG | ABS | TPU | ASA | Resin | etc.
    color           TEXT,
    brand           TEXT,
    spool_cost      INTEGER NOT NULL,  -- smallest unit
    spool_weight_g  INTEGER NOT NULL,  -- grams
    cost_per_gram   INTEGER NOT NULL,  -- smallest unit per gram (computed on insert/update)
    currency        TEXT NOT NULL DEFAULT 'PKR',
    notes           TEXT,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

-- quotes
CREATE TABLE quotes (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id),
    client_id           TEXT NOT NULL REFERENCES clients(id),
    quote_number        TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'draft', -- draft|sent|viewed|accepted|rejected|expired|converted
    issue_date          TEXT NOT NULL,
    expiry_date         TEXT,
    currency            TEXT NOT NULL DEFAULT 'PKR',
    subtotal            INTEGER NOT NULL DEFAULT 0,
    discount_type       TEXT NOT NULL DEFAULT 'none',  -- none | fixed | percent
    discount_value      INTEGER NOT NULL DEFAULT 0,
    tax_rate            INTEGER NOT NULL DEFAULT 0,    -- basis points
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    total               INTEGER NOT NULL DEFAULT 0,
    notes               TEXT,
    terms               TEXT,
    invoice_id          TEXT REFERENCES invoices(id), -- set when converted
    created_by          TEXT NOT NULL REFERENCES users(id),
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    UNIQUE(tenant_id, quote_number)
);

-- invoices
CREATE TABLE invoices (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id),
    client_id           TEXT NOT NULL REFERENCES clients(id),
    quote_id            TEXT REFERENCES quotes(id),
    invoice_number      TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'draft', -- draft|pending|sent|viewed|partially_paid|paid|overdue|cancelled|refunded
    issue_date          TEXT NOT NULL,
    due_date            TEXT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'PKR',
    subtotal            INTEGER NOT NULL DEFAULT 0,
    discount_type       TEXT NOT NULL DEFAULT 'none',
    discount_value      INTEGER NOT NULL DEFAULT 0,
    tax_rate            INTEGER NOT NULL DEFAULT 0,
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    total               INTEGER NOT NULL DEFAULT 0,
    amount_paid         INTEGER NOT NULL DEFAULT 0,
    balance_due         INTEGER NOT NULL DEFAULT 0,
    notes               TEXT,
    terms               TEXT,
    is_recurring        INTEGER NOT NULL DEFAULT 0,
    recurrence_cron     TEXT,
    watermark_text      TEXT,
    created_by          TEXT NOT NULL REFERENCES users(id),
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    UNIQUE(tenant_id, invoice_number)
);

-- line_items (shared between quotes and invoices)
CREATE TABLE line_items (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id),
    parent_type         TEXT NOT NULL,  -- 'invoice' | 'quote'
    parent_id           TEXT NOT NULL,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    item_type           TEXT NOT NULL DEFAULT 'custom', -- custom | 3d_print | filament | service
    name                TEXT NOT NULL,
    description         TEXT,
    quantity            TEXT NOT NULL DEFAULT '1',  -- stored as text to preserve decimal
    unit                TEXT,
    unit_price          INTEGER NOT NULL,
    discount_type       TEXT NOT NULL DEFAULT 'none',
    discount_value      INTEGER NOT NULL DEFAULT 0,
    tax_rate            INTEGER NOT NULL DEFAULT 0,
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    line_total          INTEGER NOT NULL,
    -- 3D print cost breakdown (nullable; only for item_type = '3d_print')
    print_filament_id   TEXT REFERENCES filament_profiles(id),
    print_grams         TEXT,           -- decimal string
    print_filament_cost INTEGER,        -- computed filament cost
    print_filament_profit_pct INTEGER,  -- basis points (200% = 20000 bp)
    print_hours         TEXT,           -- decimal string
    print_time_rate     INTEGER,        -- per hour
    print_time_cost     INTEGER,        -- computed
    print_labour_cost   INTEGER,
    print_electricity_cost INTEGER,
    print_postproc_cost INTEGER,
    print_packaging_cost INTEGER,
    print_shipping_cost INTEGER,
    print_failure_rate  INTEGER,        -- basis points
    print_profit_multiplier INTEGER,    -- basis points (e.g., 150% = 15000 bp)
    print_production_cost INTEGER,      -- total before profit
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);

-- payments
CREATE TABLE payments (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id),
    invoice_id          TEXT NOT NULL REFERENCES invoices(id),
    client_id           TEXT NOT NULL REFERENCES clients(id),
    amount              INTEGER NOT NULL,
    currency            TEXT NOT NULL,
    payment_method_id   TEXT REFERENCES payment_methods(id),
    transaction_ref     TEXT,
    notes               TEXT,
    paid_at             TEXT NOT NULL,
    created_by          TEXT NOT NULL REFERENCES users(id),
    created_at          TEXT NOT NULL
);

-- payment_methods
CREATE TABLE payment_methods (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,       -- e.g. "JazzCash", "Bank Transfer", "Cash"
    type        TEXT NOT NULL,       -- cash | bank_transfer | stripe | paypal | jazzcash | easypaisa | custom
    details     TEXT,                -- JSON: account number, IBAN, etc.
    is_active   INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- email_log
CREATE TABLE email_log (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id),
    parent_type TEXT NOT NULL,  -- 'invoice' | 'quote'
    parent_id   TEXT NOT NULL,
    to_address  TEXT NOT NULL,
    subject     TEXT NOT NULL,
    status      TEXT NOT NULL,  -- sent | failed | bounced
    error       TEXT,
    sent_at     TEXT NOT NULL
);

-- audit_log
CREATE TABLE audit_log (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id),
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    action      TEXT NOT NULL,  -- created | updated | deleted | status_changed | payment_recorded
    changed_by  TEXT NOT NULL REFERENCES users(id),
    old_value   TEXT,           -- JSON snapshot
    new_value   TEXT,           -- JSON snapshot
    changed_at  TEXT NOT NULL
);

-- tax_rates
CREATE TABLE tax_rates (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,      -- e.g. "GST 17%"
    rate        INTEGER NOT NULL,   -- basis points (17% = 1700)
    is_default  INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
```

---

## Money Handling

```go
// internal/money/money.go
// All amounts stored as int64 in smallest currency unit.
// PKR: 1 Rupee = 100 Paisa → store in paisa
// Display: divide by 100, format with locale-aware formatter.

type Money struct {
    Amount   int64
    Currency string
}

// Example: Rs 1,250.50 is stored as 125050 (paisa)
// Display helper: fmt.Sprintf("Rs %s", formatWithCommas(m.Amount, 2))
```

**Rules:**
- NEVER use `float64` for monetary calculations
- All percentage calculations use basis points (1% = 100 bp)
- Rounding: always ROUND_HALF_UP at the final display step only
- Intermediate calculations stay as integers
- Division is done last to minimise precision loss

---

## HTMX Patterns Used

```
hx-get="/invoices/{{.ID}}/line-items/new-row"   → append new line item row
hx-post="/invoices/{{.ID}}/line-items"           → save new line item
hx-put="/invoices/{{.ID}}/line-items/{{.LI.ID}}" → update line item
hx-delete="/invoices/{{.ID}}/line-items/{{.LI.ID}}" hx-confirm="..." → delete row
hx-target="#totals-block" hx-swap="outerHTML"    → live total recalculation
hx-trigger="change"                              → recalc on filament/gram change
hx-push-url="true"                               → update browser URL on navigation
hx-boost="true"                                  → on nav links for SPA-like feel
hx-indicator="#spinner"                          → loading indicator
```

---

## Chi Router Structure

```go
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.RealIP)
r.Use(SessionMiddleware)

// Public
r.Get("/login",  authHandler.ShowLogin)
r.Post("/login", authHandler.HandleLogin)
r.Post("/logout",authHandler.HandleLogout)

// Protected
r.Group(func(r chi.Router) {
    r.Use(RequireAuth)
    r.Use(LoadTenant)

    r.Get("/", dashboardHandler.Index)

    r.Route("/clients", ...)
    r.Route("/invoices", ...)
    r.Route("/quotes", ...)
    r.Route("/payments", ...)
    r.Route("/filaments", ...)
    r.Route("/settings", ...)
    r.Route("/system", ...)
    r.Get("/sse", sseHandler.Stream)
})
```

---

## PDF Generation (fpdf)

- Library: `github.com/go-pdf/fpdf`
- Template data passed as Go struct to a `GenerateInvoicePDF(inv Invoice) ([]byte, error)` function
- Layout defined in `internal/pdf/pdf.go` matching the InvoiceTemplate.md spec exactly
- Font: Noto Sans embedded as TTF bytes via `go:embed`
- Output: raw `[]byte` written to HTTP response with `Content-Type: application/pdf`

---

## Background Jobs (gocron)

```go
s := gocron.NewScheduler(time.UTC)
s.Every(1).Day().At("00:05").Do(jobs.CheckOverdueInvoices)
s.Every(1).Hour().Do(jobs.ProcessRecurringInvoices)
s.StartAsync()
```

---

## Docker Compose

```yaml
version: "3.9"
services:
  app:
    build: .
    restart: unless-stopped
    volumes:
      - ./data:/data
    environment:
      - DB_PATH=/data/layerinvoice.db
      - SESSION_SECRET=${SESSION_SECRET}
      - APP_ENV=production
    labels:
      - caddy=layerinvoice.yourdomain.com
      - caddy.reverse_proxy={{upstreams 8080}}

  caddy:
    image: lucaslorentz/caddy-docker-proxy:ci-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - caddy_data:/data

volumes:
  caddy_data:
```

---

## Go Module Dependencies

```
github.com/go-chi/chi/v5
modernc.org/sqlite
github.com/pressly/goose/v3
github.com/go-pdf/fpdf/v2
gopkg.in/gomail.v2
github.com/oklog/ulid/v2
github.com/go-co-op/gocron/v2
github.com/joho/godotenv
golang.org/x/crypto
github.com/stretchr/testify
```

No ORM. No code generator. All SQL is hand-written.
