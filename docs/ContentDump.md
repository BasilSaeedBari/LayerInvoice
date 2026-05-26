Write up an Agent.md, DESIGN.md. SKILL.md, memory.md, techstack.md, objective.md, features.md, PLAN.md. Also extend InvoiceTemplate.md

I have given a InvoiceTemplate.md, which has the signature design, you should also take the colours from it, and change design.md accordingly. Those are to be our primary colours. 

I have given a design.md, but you have to modify it to use out techstack.





Everything should be incredibly detailed, enough that the AI agent/programming, can make the full app using these files only, alone.



The way the app is supposed to work is, just like SimpleInvoice.

There should be a setting menu, where I can set the company logo, Company name, address ( not mandatory ) ; Street, City, State, Zip, Country.
It should have a place to add Email, phone number, Primary Currency of the app.
It should also have an email menu, where i can set the from address, from name, mail provider (SMTP); Host,port, user, password (invisible).
It should have an invoice settings, such as Bcc address, email subject pattern ( it extends as we change the invoice and so on ), if watermark is enabled or disabled. 

It should have an invoice menu, which has the ablity to list, create invoices.

Filament menu, to add new filament types and give its own details

Qoutes menu to list and create quotes.

Payments menu to list down all the payments made in history up till now, along with a legends at the bottom, payment methods that all your to add payment methods. 

A System menu to set tax rates, add users that can create invoices. 
Integrations is something we dont want rightnow so leave that for the feature 


It should have the clients menu where i can add a client, and then i can select a specifc cilent to see their old purchaces, income generated from them, outstanding payments, total number of quotes, invoices, outsrtadning invoiuces, paid invoices, contact information. I should also be able to edit it.
Other information i have given before ( essentiallty everything from SimpleInvoice.)

Also here is the main part, when making the invoice/quote, they way I currently do it is, that I select a filament, say it is Rs5/gram, i add a profit on the filament say 2x, hence the new price of the filament is Rs10/gram.
I then also add the print time cost, say Rs20/hour, and the calculate the total print cost. 
i currently haven't add personal labour or anything else. so the only profit i generate is from the filament.
but now this method is bad, so I want to change it to filament cost, profit on filament cost, time cost, personal labaour cost and much more, (stuff has been detailed up above).
Now genereate all the files.


Extrainfo:
| Layer      | Tech                 |
| ---------- | -------------------- |
| Backend    | Pure Go stdlib       |
| Templates  | html/template        |
| Reactive   | HTMX                 |
| CSS        | PicoCSS              |
| DB         | SQLite               |
| Queries    | raw SQL              |
| Deployment | single static binary |


<50MB RAM idle usage possible
tiny Docker images
almost instant page loads
reactive UX
minimal JS
easy self-hosting
fast development
easy debugging
easy Portainer deployment
no frontend/backend separation pain
no hydration lag
no bundle hell


Recommended Complete Stack
FINAL RECOMMENDATION
Frontend
HTMX
Alpine.js (optional)
PicoCSS
Pure CSS
Server-rendered templates
Backend
Go
Chi
sqlc
html/template or templ
net/http
Infrastructure
Docker Compose
Caddy
PostgreSQL OR SQLite
cron container
Async
SSE
background goroutines
PDFs
HTML → PDF (https://codeberg.org/go-pdf/fpdf)

Server Sent Events (SSE)

Perfect for:

invoice updates,
notification feeds,
dashboard refreshes,
job completion.

PicoCSS + custom CSS

Or:

Open Props

Or:

Pure CSS variables.

You can create a beautiful:

glassmorphism,
neumorphism,
soft-shadow,
floating UI

App Container + Mounted SQLite Volume
volumes:
  - ./data:/data

SQLite file:

Use:
net/http
Chi
sqlc
HTMX
html/template OR templ

With HTMX:

forms submit dynamically,
tables update live,
invoices auto-refresh,
dialogs load instantly,
partial HTML replaces sections.


| Tech                 | Size          |
| -------------------- | ------------- |
| HTMX                 | ~14KB gzipped |
| Alpine.js (optional) | ~15KB         |
| PicoCSS              | ~10KB         |
| Your own CSS         | maybe 5–20KB  |

| Layer                | Technology                       | Why                          |
| -------------------- | -------------------------------- | ---------------------------- |
| Backend              | Go + stdlib                      | Smallest, fastest, simplest  |
| Router               | Chi                              | Tiny and extremely fast      |
| Templates            | templ OR Go html/template        | Zero JS hydration            |
| Reactive UI          | HTMX                             | Tiny, reactive, no framework |
| Small JS enhancement | Alpine.js (optional)             | 15KB and enough              |
| CSS                  | PicoCSS OR classless CSS         | Almost zero CSS complexity   |
| Database             | PostgreSQL OR SQLite             | SQLite first, Postgres later |
| ORM                  | NONE                             | Use sqlc or raw SQL          |
| Migrations           | goose                            | Lightweight                  |
| Auth                 | lucia-like homemade session auth | Simpler than OAuth stacks    |
| Background Jobs      | gocron OR cron container         | Tiny                         |
| Email                | Gomail / SMTP                    | Simple                       |
| PDF                  | chromedp OR wkhtmltopdf          | Reliable                     |
| Realtime             | SSE (Server Sent Events)         | Lighter than websockets      |
| Deployment           | Docker Compose                   | Perfect                      |
| Reverse Proxy        | Caddy                            | Simplest TLS                 |
| Secrets              | sops/age OR Docker secrets       | Clean                        |
| IDs                  | ULID                             | Correct choice               |
| API                  | JSON REST only                   | Don’t overengineer           |
| AI Integration       | MCP server in Go                 | Easy                         |
| Frontend Build Step  | NONE                             | Critical                     |



You are building:

a fast local-first business system,
mostly CRUD + calculations,
invoices,
PDFs,
scheduling,
email,
and some reactive UI.

That means:

latency matters more than abstraction,
simplicity matters more than ecosystem size,
memory footprint matters more than “developer experience”.

Your instincts are correct:

avoid frontend frameworks,
avoid Node-heavy tooling,
avoid ORM-heavy stacks,
avoid hydration,
avoid client-side state machines,
avoid webpack/vite/bun complexity.

The fastest stack here is:

server-rendered HTML,
tiny reactive fragments,
SQLite/Postgres,
compiled backend,
no SPA,
minimal JS.


These are the features the "LayerInvoice" app should have (but you need to replace the PHP specific things with Go Specific things):
Here is a technical specification and standard documentation based on the architecture and feature set of SolidInvoice.

**1. Core Financial & Billing Engine**

* **Precision Currency Handling:** Operates using the native `Money` pattern (e.g., via MoneyPHP) instead of floating-point integers. This guarantees mathematical precision and eliminates rounding errors during currency conversions and sub-cent calculations.
* **Document State Machine:** Implements a strict state machine for document lifecycles (Draft → Pending → Paid). This enforces immutable transitions, meaning an invoice cannot be arbitrarily modified once it reaches a pending or paid state without proper reversion protocols.
* **Dynamic Tax and Discount Calculation:** Contains an algorithmic engine for localized tax rules and tiered discounting. It supports both scalar (fixed amount) and proportional (percentage) modifiers applied at either the individual line-item level or the global invoice level.
* **Automated Temporal Execution:** Relies on a scheduled background processor (CRON or message queues) to handle recurring billing. The engine dynamically clones previous invoices and dispatches them based on a flexible cron-schedule standard.
* **Headless PDF Generation:** Utilizes a rendering pipeline to convert HTML/Twig templates into branded PDF binary streams on the fly, allowing for customized layouts.

**2. Multi-Tenancy & Client Architecture**

* **ORM-Level Data Isolation:** Employs Doctrine multi-tenancy filters to create hard data boundaries. This is highly advantageous for running multiple distinct operations—such as isolating the billing pipeline of a custom manufacturing venture from an educational consultancy—on a single database instance without data bleed.
* **Granular Context Management:** Maintains distinct relational models per client, allowing for client-specific default currencies, localized addressing, and preferred communication channels.

**3. Payment Gateway Abstraction Layer**

* **Agnostic Payment Processing:** Uses `Payum` as an abstraction layer to decouple the core app from specific payment providers. This allows for drop-in integrations of diverse gateways (Stripe, PayPal, etc.) using a unified interface.
* **Stateless Payment Links:** Generates secure, single-use or session-based payment URLs that can be distributed via notifications, minimizing the need for clients to authenticate to pay.
* **PCI Compliance by Design:** The architecture ensures that PAN (Primary Account Number) data is entirely tokenized by the third-party gateway client-side. The server only handles cryptographic payment confirmations via webhooks.

**4. API, Integration & Extensibility**

* **Hypermedia REST API:** Powered by API Platform 4, exposing endpoints in highly standardized formats (JSON-LD, JSON-HAL, JSON, XML) for automated consumption.
* **Stateless Authentication:** Secures API access via `X-API-TOKEN` headers, ensuring decoupled, token-based authentication for external scripts and services.
* **Native AI Automation:** Integrates an MCP (Model Context Protocol) server. This exposes the application’s internal functions to external AI agents, allowing LLMs to trigger system actions (like drafting an invoice or pulling uncollected revenue) programmatically.
* **Event-Driven Webhooks & Notifications:** Hooks into the state machine to dispatch asynchronous events (Email, SMS, Chat) whenever a billing entity changes state.

**5. Infrastructure & Security Operations**

* **Role-Based Access Control (RBAC):** Leverages Symfony Security and Voters to strictly enforce user permissions at the controller and data-access levels.
* **Cryptographic Secrets Management:** Stores API keys, database credentials, and gateway configurations using industry-standard encryption protocols rather than plaintext environment variables.
* **Container-Native Deployment:** Fully compatible with containerized environments. Deploying this stack via Docker behind a reverse proxy like Caddy on a VPS ensures a clean, modular environment for self-hosting while abstracting SSL and routing.
* **Distributed Primary Keys:** Uses ULIDs (Universally Unique Lexicographically Sortable Identifiers) instead of auto-incrementing integers, which improves database indexing performance and prevents malicious data enumeration.

---

### Feature List Summary

If you are mapping out the checklist for your own build, here is the direct breakdown of the features you will need to implement:

**Billing & Operations**

* One-click quote-to-invoice conversion
* Recurring invoicing with customizable time schedules
* Multi-currency support with exact precision math
* Fixed and percentage-based tax and discount calculation
* Customizable PDF invoice rendering
* Strict invoice state transitions (Draft, Pending, Paid, Canceled)

**Client Management**

* Centralized client and contact CRM
* Per-client configuration (currency, language, billing address)
* Multi-tenant company management from a single installation

**Payments**

* Plug-and-play architecture for Stripe, PayPal, and custom gateways
* Auto-generated online payment links attached to invoices
* Zero-server-touch PCI-compliant payment flows

**API & Automation**

* Full CRUD REST API supporting multiple JSON schemas
* Token-based API authentication
* MCP server implementation for direct AI-agent interaction
* Multi-channel notification dispatch (Email, SMS, Webhooks)

**Security & Platform**

* Role-based user permissions and access control
* Encrypted storage for external secrets and API keys
* Self-hostable containerized deployment support
* ULID-based database primary keys



List of things in the app:
# Minimal Yet Complete Feature Outline For Your App

Your app should NOT become:

* bloated ERP software,
* accounting software,
* or enterprise management software.

It should stay focused on:

* invoices,
* quotes,
* payments,
* clients,
* automation,
* and 3D printing workflows.

The goal is:

> “Get quote → send invoice → get paid → track status”
> with the least friction possible.

The features below are split into:

* MVP Core
* Strong Production Features
* 3D Printing Specific Features
* Nice-to-have Future Features

Based on:

* [SolidInvoice GitHub](https://github.com/SolidInvoice/SolidInvoice?utm_source=chatgpt.com)
* [SolidInvoice Features](https://solidinvoice.co/features?utm_source=chatgpt.com)
* [SolidInvoice Docs](https://solidinvoice.co/docs/?utm_source=chatgpt.com)

([GitHub][1])

---

# 1. CORE DASHBOARD SYSTEM

## Main Dashboard

The homepage after login.

Should show:

* unpaid invoices
* overdue invoices
* pending quotes
* recent payments
* revenue this month
* quote conversion rate
* recurring invoices due
* recently active clients

For 3D printing:

* filament consumed this month
* most used filament
* print hours billed
* profit estimates

---

# 2. CLIENT MANAGEMENT SYSTEM

## Client Database

Every customer should have:

### Basic Information

* company name
* person name
* email
* phone number
* WhatsApp number
* website
* tax ID / NTN / VAT

### Billing Information

* billing address
* shipping address
* preferred currency
* payment terms
* default tax rate
* default discount
* preferred payment method

### Client Metadata

* notes
* tags
* internal comments
* client category
* active/inactive status

### History

* quotes history
* invoice history
* payment history
* email history

---

# 3. QUOTES / ESTIMATES SYSTEM

This is one of the MOST important sections.

## Quote Creation

A quote should support:

* quote number
* creation date
* expiry date
* client selection
* seller/company selection
* currency
* tax configuration
* discount configuration
* notes
* terms & conditions

---

## Quote Line Items

Each line item should support:

* item name
* description
* quantity
* unit
* unit price
* tax
* subtotal
* discount
* total

---

## Quote Statuses

* Draft
* Sent
* Viewed
* Accepted
* Rejected
* Expired
* Converted

---

## Quote Actions

* duplicate quote
* export PDF
* print quote
* email quote
* generate share link
* convert to invoice
* archive
* soft delete

---

## Quote Templates

Different layouts:

* modern
* minimal
* industrial
* compact
* premium

---

# 4. INVOICE SYSTEM

This is the core of the entire app.

---

## Invoice Creation

Invoices should support:

* invoice number
* client
* seller/company
* issue date
* due date
* currency
* payment terms
* taxes
* discounts
* notes
* attachments

---

## Invoice States

You MUST implement a strict state system.

Recommended:

```text
Draft
Pending
Sent
Viewed
Partially Paid
Paid
Overdue
Canceled
Refunded
```

Once:

* Paid
* Refunded

the invoice should become immutable unless reverted.

---

## Invoice Actions

* create
* duplicate
* archive
* edit
* mark paid
* partial payment
* refund
* convert to recurring
* export PDF
* print
* email
* generate payment link

---

## Invoice Editing Rules

This matters A LOT.

Recommended behavior:

### Draft

fully editable

### Sent

editable with revision tracking

### Paid

locked unless admin override

---

# 5. 3D PRINTING SPECIFIC ENGINE

THIS is what makes your app unique.

---

# Print Cost Calculator

## Inputs

* print name
* STL/project
* grams used
* filament profile
* print time
* machine used
* electricity rate
* labor cost
* post-processing cost
* packaging cost
* shipping cost
* failure rate %
* profit multiplier

---

## Filament Profiles

Each filament should support:

* filament name
* material type
* color
* spool cost
* spool weight
* cost per gram
* vendor
* notes

---

## Automatic Calculations

The app should calculate:

* filament cost
* electricity cost
* machine time cost
* labor cost
* total production cost
* target profit
* final customer price

---

## Saved Presets

Allow:

* recurring print presets
* reusable product templates
* saved print profiles

---

# 6. PAYMENTS SYSTEM

## Payment Methods

* bank transfer
* cash
* Stripe
* PayPal
* manual/offline

---

## Payment Tracking

Track:

* amount paid
* date paid
* transaction ID
* payment note
* payment method

---

## Partial Payments

Critical feature.

Example:

* invoice = $100
* client pays $40
* remaining = $60

---

## Automatic Status Updates

Invoice auto changes:

* pending → paid
* pending → partially paid
* overdue → paid

---

# 7. PDF GENERATION SYSTEM

Extremely important.

---

## PDF Features

* logo
* company branding
* accent colors
* signatures
* QR code
* payment instructions
* bank details
* terms
* watermark

---

## PDF Templates

You NEED multiple templates.

Examples:

| Template   | Style            |
| ---------- | ---------------- |
| Minimal    | clean            |
| Modern     | glassmorphism    |
| Industrial | technical        |
| Compact    | small businesses |
| Corporate  | traditional      |

---

# 8. EMAIL SYSTEM

## Email Sending

Should support:

* quote emails
* invoice emails
* reminders
* payment confirmations
* overdue notices

---

## Email Templates

Customizable HTML templates.

Variables:

* client name
* invoice number
* amount
* payment link
* due date

---

## Email Logging

Track:

* sent date
* opened
* bounced
* failed

---

# 9. RECURRING BILLING

Very important.

---

## Recurring Invoices

Allow:

* weekly
* monthly
* yearly
* custom cron

---

## Automation

Automatically:

* generate invoice
* email invoice
* send reminder

---

# 10. NOTIFICATIONS SYSTEM

## Internal Notifications

* invoice paid
* overdue invoice
* quote accepted
* recurring invoice created

---

## External Notifications

* email

---

# 11. MULTI-COMPANY / MULTI-TENANT

You specifically mentioned:

* printer sales
* print service business

So you NEED this.

---

## Companies

Support:

* multiple businesses
* separate branding
* separate invoices
* separate clients
* separate bank accounts

---

## Tenant Isolation

Everything should belong to:

```text
tenant_id
```

---

# 12. USER SYSTEM

## Authentication

* login
* logout
* password reset
* session management

---

## Roles

You probably only need:

| Role     | Access           |
| -------- | ---------------- |
| Owner    | full             |
| Employee | invoice creation |
| Viewer   | read-only        |

Avoid enterprise RBAC complexity initially.

---

# 13. SETTINGS PANEL

## Business Settings

* company name
* logo
* address
* email
* phone
* website

---

## Invoice Settings

* invoice numbering
* prefixes
* tax defaults
* due date defaults

---

## Currency Settings

* currency
* decimal precision
* rounding mode

---

## Email Settings

* SMTP
* sender email
* signatures

---

# 14. SEARCH & FILTERING

You NEED this.

Search:

* invoices
* quotes
* clients
* products

Filter:

* paid
* overdue
* draft
* client
* date
* amount

---

# 15. FILE ATTACHMENTS

Allow attaching:

* STL files
* screenshots
* reference images
* receipts
* PDFs

---

# 16. API SYSTEM

Minimal REST API only.

---

## API Features

* token auth
* CRUD
* webhooks
* invoice creation
* payment updates

---

# 17. AUDIT / HISTORY SYSTEM

Critical for invoices.

Track:

* who edited
* what changed
* when changed

Especially:

* paid invoices
* totals
* taxes

---

# 18. BACKUPS & EXPORTS

## Export

* CSV
* JSON
* PDF

---

## Backups

* scheduled DB backup
* manual export

---

# 19. IMPORT SYSTEM

Useful feature.

Import:

* clients
* invoices
* products

from:

* CSV
* JSON

---

# 20. MODERN UI FEATURES

You specifically want:

* bubble
* soft
* modern
* reactive
* lightweight

So:

## UI Features

* instant search
* HTMX partial reloads
* inline editing
* modal forms
* command palette
* dark mode
* responsive layout

---

# 21. FEATURES YOU SHOULD AVOID INITIALLY

These will massively slow development:

* inventory ERP
* accounting ledger system
* payroll
* blockchain nonsense
* AI chatbots
* mobile apps
* websocket-heavy realtime
* full bookkeeping
* gigantic plugin systems

---

# ABSOLUTE MVP FEATURE SET

If you want the TRUE minimal first version:

## MVP

### Clients

* add/edit/delete clients

### Quotes

* create quotes
* PDF export
* email
* convert to invoice

### Invoices

* create/edit invoices
* status system
* payments
* PDF
* email

### 3D Printing

* filament profiles
* cost calculator
* automatic pricing

### Settings

* seller info
* bank info
* templates
* taxes

### Infrastructure

* auth
* Docker compose
* SQLite/Postgres
* backups

That alone is already a very strong v1.

---

# BEST DEVELOPMENT ORDER

## Phase 1

* auth
* DB
* clients
* invoice CRUD

## Phase 2

* quotes
* PDFs
* email

## Phase 3

* 3D print calculator
* templates
* recurring invoices

## Phase 4

* API
* multi-tenant
* payment integrations

## Phase 5

* automation
* notifications
* analytics

---

# Most Important Design Principle

DO NOT build:
> “a software suite”

Build:
> “the fastest way from quote to payment.”
That focus is exactly why minimalist invoicing apps succeed.
[1]: https://github.com/SolidInvoice/SolidInvoice?utm_source=chatgpt.com "GitHub - SolidInvoice/SolidInvoice: Simple and elegant invoicing solution."

The user wants me to create multiple documentation files for a Go-based invoicing application called "LayerInvoice". Let me analyze all the provided documents and create comprehensive files. Let me plan what each file needs:

Agent.md - Instructions for AI agent to build the app
DESIGN.md - Design system based on InvoiceTemplate.md colors + Sentri-inspired design adapted for Go/HTMX stack
SKILL.md - Skills/capabilities needed
memory.md - Context memory for the agent
techstack.md - Go + HTMX + PicoCSS + SQLite stack details
objective.md - App objective and goals
features.md - Detailed feature list
PLAN.md - Development plan/phases
InvoiceTemplate.md (extended) - Extended invoice template

