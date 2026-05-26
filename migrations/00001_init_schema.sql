-- +goose Up
-- +goose StatementBegin

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
    tenant_id       TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
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
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    PRIMARY KEY (tenant_id, key)
);

-- clients
CREATE TABLE clients (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
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
    tenant_id       TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
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
    tenant_id           TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id           TEXT NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
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
    invoice_id          TEXT, -- set when converted, references invoices(id) but handled in application layer to prevent cyclic reference cycles on table create
    created_by          TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    UNIQUE(tenant_id, quote_number)
);

-- invoices
CREATE TABLE invoices (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id           TEXT NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    quote_id            TEXT REFERENCES quotes(id) ON DELETE SET NULL,
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
    created_by          TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    UNIQUE(tenant_id, invoice_number)
);

-- line_items (shared between quotes and invoices)
CREATE TABLE line_items (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_type         TEXT NOT NULL,  -- 'invoice' | 'quote'
    parent_id           TEXT NOT NULL,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    item_type           TEXT NOT NULL DEFAULT 'custom', -- custom | 3d_print | filament | service
    name                TEXT NOT NULL,
    description         TEXT,
    quantity            TEXT NOT NULL DEFAULT '1',  -- decimal string
    unit                TEXT,
    unit_price          INTEGER NOT NULL,
    discount_type       TEXT NOT NULL DEFAULT 'none',
    discount_value      INTEGER NOT NULL DEFAULT 0,
    tax_rate            INTEGER NOT NULL DEFAULT 0,
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    line_total          INTEGER NOT NULL,
    -- 3D print cost breakdown
    print_filament_id   TEXT REFERENCES filament_profiles(id) ON DELETE SET NULL,
    print_grams         TEXT,           -- decimal string
	print_filament_cost INTEGER,
    print_filament_profit_pct INTEGER,  -- basis points
    print_hours         TEXT,           -- decimal string
    print_time_rate     INTEGER,        -- per hour
    print_time_cost     INTEGER,        -- computed
    print_labour_cost   INTEGER,
    print_electricity_cost INTEGER,
    print_postproc_cost INTEGER,
    print_packaging_cost INTEGER,
    print_shipping_cost INTEGER,
    print_failure_rate  INTEGER,        -- basis points
    print_profit_multiplier INTEGER,    -- basis points
    print_production_cost INTEGER,      -- total before profit
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);

-- payment_methods
CREATE TABLE payment_methods (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,       -- e.g. "JazzCash", "Bank Transfer", "Cash"
    type        TEXT NOT NULL,       -- cash | bank_transfer | stripe | paypal | jazzcash | easypaisa | custom
    details     TEXT,                -- JSON: account number, IBAN, etc.
    is_active   INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- payments
CREATE TABLE payments (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id          TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    client_id           TEXT NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    amount              INTEGER NOT NULL,
    currency            TEXT NOT NULL,
    payment_method_id   TEXT REFERENCES payment_methods(id) ON DELETE SET NULL,
    transaction_ref     TEXT,
    notes               TEXT,
    paid_at             TEXT NOT NULL,
    created_by          TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          TEXT NOT NULL
);

-- email_log
CREATE TABLE email_log (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
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
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    action      TEXT NOT NULL,  -- created | updated | deleted | status_changed | payment_recorded
    changed_by  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    old_value   TEXT,           -- JSON snapshot
    new_value   TEXT,           -- JSON snapshot
    changed_at  TEXT NOT NULL
);

-- tax_rates
CREATE TABLE tax_rates (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,      -- e.g. "GST 17%"
    rate        INTEGER NOT NULL,   -- basis points (17% = 1700)
    is_default  INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tax_rates;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS email_log;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS line_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS quotes;
DROP TABLE IF EXISTS filament_profiles;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
-- +goose StatementEnd
