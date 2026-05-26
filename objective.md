# LayerInvoice — Objective

## One-Line Mission

> "The fastest path from 3D print cost estimate to paid invoice — with zero frontend complexity."

---

## What LayerInvoice Is

LayerInvoice is a **self-hosted, Go-native invoicing and quoting application** built specifically for a 3D printing service business. It is opinionated, minimal, and fast. It handles the full billing lifecycle: cost calculation → quote → invoice → payment tracking.

It is NOT accounting software. It is NOT an ERP. It is NOT a general-purpose SaaS.

---

## Core Problem It Solves

The owner of a 3D printing business currently calculates job cost manually:
- Filament cost per gram × grams used
- A profit multiplier on filament
- Print time cost

This is error-prone, underdocumented, and disconnected from client invoicing. The goal is to unify the cost engine with the invoicing system so that a quote is generated directly from a 3D print job estimate — no spreadsheet, no copy-pasting, no manual math.

---

## Primary User

A single-operator or small-team 3D printing service business. The operator:
- Runs one or two physical businesses (multi-tenant)
- Accepts orders via WhatsApp, email, or walk-in
- Needs to generate professional quotes and invoices quickly
- Tracks who has paid, who owes, and which jobs are pending
- Works primarily from a desktop browser on a self-hosted VPS or local server

---

## Design Philosophy

1. **Speed over features.** Every click removed from quote-to-invoice is more valuable than any new feature.
2. **No frontend frameworks.** Server-rendered HTML + HTMX partial updates only. No React, no Vue, no Svelte.
3. **One binary.** The entire application compiles to a single static Go binary. No Node, no Python, no runtime dependencies.
4. **SQLite first.** The database is a single file on disk. Postgres migration is a future upgrade path, not a day-one concern.
5. **Self-hostable.** Docker Compose + Caddy. The owner controls their data.
6. **Precision math.** All monetary values are stored as integers (smallest currency unit, e.g., paisa for PKR). No floating-point in financial calculations, ever.

---

## What Success Looks Like

- Operator opens the app, selects a filament profile, enters grams and print time, and the system produces a fully costed line item.
- That line item is added to a quote, which is emailed to the client in under 2 minutes.
- Client accepts; operator converts quote to invoice in one click.
- Payment is recorded; invoice status updates automatically.
- At month-end, operator can see total revenue, outstanding amounts, and filament consumed — from the dashboard without running a single query.

---

## Out of Scope (v1)

- Inventory / stock tracking
- Payroll
- Full double-entry accounting ledger
- Mobile native apps
- Public-facing client portal
- WebSocket real-time (SSE only for relevant events)
- Plugin system
- AI chatbot integration
- Blockchain or crypto payments
