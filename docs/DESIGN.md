# LayerInvoice — Design System

## Design Philosophy

LayerInvoice is a **server-rendered, HTMX-powered Go application**. All design is expressed via:
1. PicoCSS v2 (classless semantic HTML baseline)
2. A single `static/css/app.css` file of CSS custom properties (design tokens) layered on top
3. Semantic HTML that maps naturally to PicoCSS's component styles
4. No utility class frameworks. No Tailwind. No component libraries.

The visual language is **clean, professional, and document-centric** — designed to feel like a premium PDF invoice come to life in a web interface. It uses the exact color palette established in the InvoiceTemplate.md specification.

---

## Color Palette (from InvoiceTemplate.md — canonical source)

These are the immutable brand colors. They must not be changed or extended without updating both `app.css` and `InvoiceTemplate.md`.

| Token                    | Hex       | Role                                                      |
| ------------------------ | --------- | --------------------------------------------------------- |
| `--color-text-primary`   | `#333333` | Major headings, totals, primary data, body text           |
| `--color-text-secondary` | `#777777` | Labels, contact info, helper text, table meta             |
| `--color-surface-alt`    | `#F9F9F9` | Table header rows, sidebar background, input backgrounds  |
| `--color-border`         | `#EAEAEA` | Table row dividers, card borders, section separations     |
| `--color-background`     | `#FFFFFF` | Page background, card surfaces, modal backgrounds         |

### Semantic Tokens (derived from palette)

```css
/* Derived from InvoiceTemplate.md base palette */
--color-text-primary:     #333333;
--color-text-secondary:   #777777;
--color-text-muted:       #AAAAAA;  /* lighter than secondary; placeholder text */
--color-text-inverse:     #FFFFFF;  /* text on dark surfaces */
--color-surface:          #FFFFFF;  /* main page/card background */
--color-surface-alt:      #F9F9F9;  /* table headers, input bg, sidebar */
--color-surface-hover:    #F2F2F2;  /* row hover state */
--color-border:           #EAEAEA;  /* standard dividers */
--color-border-strong:    #CCCCCC;  /* stronger separators, focus rings */
--color-accent:           #333333;  /* primary action color (dark, document-inspired) */
--color-accent-hover:     #222222;
--color-accent-light:     #F9F9F9;  /* tinted surface for accent context */

/* Status Colors */
--color-status-draft:         #777777;
--color-status-draft-bg:      #F9F9F9;
--color-status-pending:       #D97706;   /* amber */
--color-status-pending-bg:    #FFFBEB;
--color-status-sent:          #2563EB;   /* blue */
--color-status-sent-bg:       #EFF6FF;
--color-status-paid:          #16A34A;   /* green */
--color-status-paid-bg:       #F0FDF4;
--color-status-overdue:       #DC2626;   /* red */
--color-status-overdue-bg:    #FEF2F2;
--color-status-partial:       #7C3AED;   /* purple */
--color-status-partial-bg:    #F5F3FF;
--color-status-cancelled:     #6B7280;
--color-status-cancelled-bg:  #F3F4F6;
--color-status-refunded:      #0891B2;
--color-status-refunded-bg:   #ECFEFF;
--color-status-accepted:      #16A34A;
--color-status-accepted-bg:   #F0FDF4;
--color-status-rejected:      #DC2626;
--color-status-rejected-bg:   #FEF2F2;
--color-status-expired:       #6B7280;
--color-status-expired-bg:    #F3F4F6;
--color-status-converted:     #2563EB;
--color-status-converted-bg:  #EFF6FF;
--color-status-viewed:        #0891B2;
--color-status-viewed-bg:     #ECFEFF;

/* Feedback */
--color-success:    #16A34A;
--color-success-bg: #F0FDF4;
--color-warning:    #D97706;
--color-warning-bg: #FFFBEB;
--color-error:      #DC2626;
--color-error-bg:   #FEF2F2;
--color-info:       #2563EB;
--color-info-bg:    #EFF6FF;
```

---

## Typography (from InvoiceTemplate.md — canonical source)

**Primary Font:** Noto Sans (Google Fonts)  
**CDN:** `https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;600;700&display=swap`  
**Fallback stack:** `'Noto Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif`

| Token                     | Size  | Weight | Use                                            |
| ------------------------- | ----- | ------ | ---------------------------------------------- |
| `--font-size-hero`        | 24pt  | 700    | Page title "INVOICE", major document headings  |
| `--font-size-section`     | 16pt  | 600    | Section headings, card titles                  |
| `--font-size-subheading`  | 12pt  | 600    | Client name, important labels                  |
| `--font-size-body`        | 10pt  | 400    | Standard body text, table cells                |
| `--font-size-label`       | 10pt  | 400    | Secondary labels, meta info                    |
| `--font-size-small`       | 9pt   | 400    | Fine print, table column sub-descriptions      |
| `--font-size-micro`       | 8pt   | 400    | Watermarks, legal footnotes                    |

In CSS (screen rendering converts pt → px at 1pt ≈ 1.333px):

```css
--text-hero:       2rem;    /* ~24pt */
--text-section:    1.25rem; /* ~16pt */
--text-subheading: 1rem;    /* ~12pt; up-weights to 600 */
--text-body:       0.875rem;/* ~10pt */
--text-label:      0.875rem;/* ~10pt */
--text-small:      0.8125rem;/* ~9pt */
--text-micro:      0.75rem; /* ~8pt */
```

---

## Spacing & Layout

PicoCSS base unit is `1rem` (16px). LayerInvoice uses the following spacing scale mapped to the document's 0.75-inch margin standard:

```css
--space-1:  0.25rem;  /* 4px  */
--space-2:  0.5rem;   /* 8px  */
--space-3:  0.75rem;  /* 12px */
--space-4:  1rem;     /* 16px */
--space-5:  1.25rem;  /* 20px */
--space-6:  1.5rem;   /* 24px */
--space-8:  2rem;     /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
--space-16: 4rem;     /* 64px */
```

### Application Layout

```
┌─────────────────────────────────────────────────────────┐
│ TOP NAV (64px)  Logo | Company Name         User + Menu │
├──────────┬──────────────────────────────────────────────┤
│          │                                              │
│ SIDEBAR  │  MAIN CONTENT AREA                           │
│ (240px)  │  max-width: 1200px; padding: 2rem            │
│          │                                              │
│ Nav      │  ┌─────────────────────────────────────────┐ │
│ Links    │  │  PAGE HEADER                            │ │
│          │  │  Title + Actions (buttons right-aligned)│ │
│          │  └─────────────────────────────────────────┘ │
│          │                                              │
│          │  ┌─────────────────────────────────────────┐ │
│          │  │  CONTENT CARD(s)                        │ │
│          │  │  Background: #FFFFFF                    │ │
│          │  │  Border: 1px solid #EAEAEA              │ │
│          │  │  Border-radius: 8px                     │ │
│          │  │  Padding: 1.5rem                        │ │
│          │  └─────────────────────────────────────────┘ │
├──────────┴──────────────────────────────────────────────┤
│ FOOTER (optional, minimal)                              │
└─────────────────────────────────────────────────────────┘
```

### Sidebar Navigation Structure

```
┌────────────────────────┐
│ 🖨  LayerInvoice        │  ← Logo + app name
├────────────────────────┤
│ ◉  Dashboard           │
├────────────────────────┤
│ BILLING                │  ← section label (uppercase, #777777, 9pt)
│    Invoices            │
│    Quotes              │
│    Payments            │
├────────────────────────┤
│ PRINT SHOP             │
│    Filaments           │
├────────────────────────┤
│ CRM                    │
│    Clients             │
├────────────────────────┤
│ ADMIN                  │
│    Settings            │
│    System              │
└────────────────────────┘
```

---

## Components

All components are **semantic HTML** styled via CSS custom properties. No CSS classes where the PicoCSS default covers it. Custom classes are in `app.css` only.

### Cards

```html
<!-- Standard content card -->
<article class="li-card">
  <header class="li-card__header">
    <h2>Section Title</h2>
    <div class="li-card__actions">
      <a href="..." role="button" class="li-btn li-btn--primary">New Invoice</a>
    </div>
  </header>
  <!-- content -->
</article>
```

```css
.li-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: var(--space-6);
  margin-bottom: var(--space-6);
}
.li-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
```

### Buttons

LayerInvoice has 3 button variants. They map to semantic `<a role="button">`, `<button>`, and `<input type="submit">`.

```css
/* Primary: dark, document-inspired */
.li-btn--primary {
  background-color: var(--color-accent);       /* #333333 */
  color: var(--color-text-inverse);            /* #FFFFFF */
  border: none;
  padding: var(--space-2) var(--space-4);
  border-radius: 6px;
  font-family: 'Noto Sans', sans-serif;
  font-weight: 600;
  font-size: var(--text-body);
  cursor: pointer;
  text-decoration: none;
  display: inline-block;
}
.li-btn--primary:hover {
  background-color: var(--color-accent-hover); /* #222222 */
}

/* Secondary: bordered outline */
.li-btn--secondary {
  background-color: transparent;
  color: var(--color-text-primary);
  border: 1px solid var(--color-border-strong);
  padding: var(--space-2) var(--space-4);
  border-radius: 6px;
  font-weight: 600;
  font-size: var(--text-body);
}

/* Ghost: text-only */
.li-btn--ghost {
  background: none;
  border: none;
  color: var(--color-text-secondary);
  padding: var(--space-2) var(--space-3);
  cursor: pointer;
  font-size: var(--text-body);
}
.li-btn--ghost:hover {
  color: var(--color-text-primary);
}

/* Danger */
.li-btn--danger {
  background-color: var(--color-error);
  color: var(--color-text-inverse);
  border: none;
}

/* Sizes */
.li-btn--sm { padding: var(--space-1) var(--space-3); font-size: var(--text-small); }
.li-btn--lg { padding: var(--space-3) var(--space-6); font-size: var(--text-subheading); }
```

### Status Badges

```html
<span class="li-badge li-badge--paid">Paid</span>
<span class="li-badge li-badge--draft">Draft</span>
<span class="li-badge li-badge--overdue">Overdue</span>
```

```css
.li-badge {
  display: inline-block;
  padding: 2px var(--space-2);
  border-radius: 4px;
  font-size: var(--text-small);
  font-weight: 600;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}
.li-badge--draft     { color: var(--color-status-draft);     background: var(--color-status-draft-bg); }
.li-badge--pending   { color: var(--color-status-pending);   background: var(--color-status-pending-bg); }
.li-badge--sent      { color: var(--color-status-sent);      background: var(--color-status-sent-bg); }
.li-badge--paid      { color: var(--color-status-paid);      background: var(--color-status-paid-bg); }
.li-badge--overdue   { color: var(--color-status-overdue);   background: var(--color-status-overdue-bg); }
.li-badge--partial   { color: var(--color-status-partial);   background: var(--color-status-partial-bg); }
.li-badge--cancelled { color: var(--color-status-cancelled); background: var(--color-status-cancelled-bg); }
.li-badge--refunded  { color: var(--color-status-refunded);  background: var(--color-status-refunded-bg); }
.li-badge--accepted  { color: var(--color-status-accepted);  background: var(--color-status-accepted-bg); }
.li-badge--rejected  { color: var(--color-status-rejected);  background: var(--color-status-rejected-bg); }
.li-badge--expired   { color: var(--color-status-expired);   background: var(--color-status-expired-bg); }
.li-badge--converted { color: var(--color-status-converted); background: var(--color-status-converted-bg); }
.li-badge--viewed    { color: var(--color-status-viewed);    background: var(--color-status-viewed-bg); }
```

### Tables

```html
<div class="li-table-wrap">
  <table>
    <thead>
      <tr>
        <th>Invoice #</th>
        <th>Client</th>
        <th>Date</th>
        <th class="li-col-right">Total</th>
        <th>Status</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td><a href="/invoices/01JX...">#INV-0001</a></td>
        <td>Acme Corp</td>
        <td>01/06/2025</td>
        <td class="li-col-right li-money">Rs 12,500</td>
        <td><span class="li-badge li-badge--paid">Paid</span></td>
        <td class="li-col-actions">
          <a href="...">View</a>
          <a href="...">PDF</a>
        </td>
      </tr>
    </tbody>
  </table>
</div>
```

```css
/* PicoCSS handles table base styles. Custom overrides: */
table thead tr {
  background: var(--color-surface-alt);  /* #F9F9F9 */
}
table thead th {
  font-size: var(--text-small);
  font-weight: 600;
  color: var(--color-text-primary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  padding: 10px 15px;
}
table tbody tr {
  border-bottom: 1px solid var(--color-border);
}
table tbody tr:hover {
  background: var(--color-surface-hover);
}
table tbody td {
  padding: 12px 15px;
  font-size: var(--text-body);
  color: var(--color-text-primary);
}
.li-col-right { text-align: right; }
.li-money     { font-feature-settings: "tnum"; font-variant-numeric: tabular-nums; }
.li-col-actions { display: flex; gap: var(--space-2); justify-content: flex-end; }
```

### Forms

PicoCSS handles all form field styling. Custom overrides:

```css
input, select, textarea {
  background: var(--color-surface-alt);  /* #F9F9F9 */
  border: 1px solid var(--color-border); /* #EAEAEA */
  color: var(--color-text-primary);      /* #333333 */
  border-radius: 6px;
  font-family: 'Noto Sans', sans-serif;
  font-size: var(--text-body);
}
input:focus, select:focus, textarea:focus {
  border-color: var(--color-border-strong);
  outline: none;
  box-shadow: 0 0 0 3px rgba(51, 51, 51, 0.08);
}
label {
  color: var(--color-text-secondary);  /* #777777 */
  font-size: var(--text-small);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: var(--space-1);
  display: block;
}
```

### Invoice Line Item Row (HTMX-driven)

```html
<!-- Rendered via HTMX partial -->
<tr class="li-line-item" id="li-row-{{.ID}}">
  <td>
    <input type="text" name="name" value="{{.Name}}" placeholder="Item name" required>
    <input type="text" name="description" value="{{.Description}}" placeholder="Description (optional)">
  </td>
  <td><input type="number" name="qty" value="{{.Quantity}}" step="0.01" min="0"
       hx-trigger="change" hx-post="/invoice/{{$.InvoiceID}}/li/{{.ID}}/calc"
       hx-target="#totals-block" hx-include="closest tr"></td>
  <td><input type="text" name="unit" value="{{.Unit}}" placeholder="pcs"></td>
  <td><input type="number" name="unit_price" value="{{.UnitPriceDisplay}}" step="0.01"
       hx-trigger="change" hx-post="/invoice/{{$.InvoiceID}}/li/{{.ID}}/calc"
       hx-target="#totals-block" hx-include="closest tr"></td>
  <td class="li-col-right li-money">{{.LineTotalDisplay}}</td>
  <td>
    <button class="li-btn li-btn--ghost li-btn--sm"
            hx-delete="/invoice/{{$.InvoiceID}}/li/{{.ID}}"
            hx-target="closest tr" hx-swap="outerHTML">✕</button>
  </td>
</tr>
```

### 3D Print Cost Modal

When the user adds a line item of type `3d_print`, a modal expands with all cost inputs:

```html
<dialog id="print-calc-modal" open>
  <article>
    <header>
      <button class="li-btn li-btn--ghost" onclick="this.closest('dialog').close()">✕</button>
      <h3>3D Print Cost Calculator</h3>
    </header>
    <form hx-post="/invoice/{{.ID}}/li/new-print" hx-target="#line-items-tbody"
          hx-swap="beforeend" hx-on::after-request="this.closest('dialog').close()">
      
      <section class="li-calc-section">
        <h4>Filament</h4>
        <div class="li-form-row">
          <label>Filament Profile
            <select name="filament_id" hx-get="/filaments/cost-per-gram"
                    hx-trigger="change" hx-target="#cpg-display" hx-include="this">
              {{range .Filaments}}<option value="{{.ID}}">{{.Name}} (Rs{{.CostPerGramDisplay}}/g)</option>{{end}}
            </select>
          </label>
          <label>Grams Used
            <input type="number" name="grams" step="0.1" min="0" placeholder="0.0"
                   hx-trigger="change" hx-post="/calc/print-cost"
                   hx-include="closest form" hx-target="#calc-preview">
          </label>
          <label>Profit on Filament
            <input type="number" name="filament_profit_pct" value="100" step="1" min="0"
                   placeholder="100" hx-trigger="change" hx-post="/calc/print-cost"
                   hx-include="closest form" hx-target="#calc-preview">
            <small>% markup (100% = 2× cost)</small>
          </label>
        </div>
      </section>

      <section class="li-calc-section">
        <h4>Time & Labour</h4>
        <div class="li-form-row">
          <label>Print Hours
            <input type="number" name="print_hours" step="0.1" min="0" placeholder="0.0"
                   hx-trigger="change" hx-post="/calc/print-cost"
                   hx-include="closest form" hx-target="#calc-preview">
          </label>
          <label>Time Rate (Rs/hr)
            <input type="number" name="time_rate" step="1" min="0" placeholder="20"
                   hx-trigger="change" hx-post="/calc/print-cost"
                   hx-include="closest form" hx-target="#calc-preview">
          </label>
          <label>Labour Cost (Rs)
            <input type="number" name="labour_cost" step="1" min="0" placeholder="0"
                   hx-trigger="change" hx-post="/calc/print-cost"
                   hx-include="closest form" hx-target="#calc-preview">
          </label>
        </div>
      </section>

      <section class="li-calc-section">
        <h4>Additional Costs</h4>
        <div class="li-form-row">
          <label>Electricity (Rs)<input type="number" name="electricity_cost" step="1" min="0"></label>
          <label>Post-Processing (Rs)<input type="number" name="postproc_cost" step="1" min="0"></label>
          <label>Packaging (Rs)<input type="number" name="packaging_cost" step="1" min="0"></label>
          <label>Shipping (Rs)<input type="number" name="shipping_cost" step="1" min="0"></label>
        </div>
      </section>

      <section class="li-calc-section">
        <h4>Profit & Risk</h4>
        <div class="li-form-row">
          <label>Failure Rate %
            <input type="number" name="failure_rate" step="0.5" min="0" max="100" placeholder="5">
          </label>
          <label>Final Profit Multiplier %
            <input type="number" name="profit_multiplier" step="5" min="0" placeholder="0">
            <small>Applied after all costs. 50% = 1.5× total cost.</small>
          </label>
        </div>
      </section>

      <!-- Live preview updated by HTMX -->
      <div id="calc-preview" class="li-calc-preview">
        <!-- Rendered server-side partial with breakdown table -->
      </div>

      <footer>
        <button type="submit" class="li-btn li-btn--primary">Add to Invoice</button>
        <button type="button" class="li-btn li-btn--secondary" onclick="this.closest('dialog').close()">Cancel</button>
      </footer>
    </form>
  </article>
</dialog>
```

### Dashboard Stat Cards

```html
<div class="li-stat-grid">
  <div class="li-stat-card">
    <p class="li-stat-card__label">Unpaid Invoices</p>
    <p class="li-stat-card__value">Rs 45,200</p>
    <p class="li-stat-card__sub">3 invoices</p>
  </div>
  <!-- repeat -->
</div>
```

```css
.li-stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-4);
  margin-bottom: var(--space-8);
}
.li-stat-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: var(--space-5);
}
.li-stat-card__label {
  font-size: var(--text-small);
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
  margin-bottom: var(--space-2);
}
.li-stat-card__value {
  font-size: var(--text-section);
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}
.li-stat-card__sub {
  font-size: var(--text-small);
  color: var(--color-text-muted);
  margin: var(--space-1) 0 0;
}
```

### Flash / Toast Messages

```html
<!-- Appended to DOM via HTMX HX-Trigger header -->
<div class="li-flash li-flash--success" id="flash-msg"
     x-data="{show: true}" x-show="show"
     x-init="setTimeout(() => show = false, 4000)">
  ✓ Invoice created successfully.
</div>
```

```css
.li-flash {
  position: fixed;
  top: var(--space-4);
  right: var(--space-4);
  padding: var(--space-3) var(--space-5);
  border-radius: 8px;
  font-size: var(--text-body);
  font-weight: 600;
  z-index: 9999;
  box-shadow: 0 4px 16px rgba(0,0,0,0.12);
}
.li-flash--success { background: var(--color-success-bg); color: var(--color-success); border: 1px solid var(--color-success); }
.li-flash--error   { background: var(--color-error-bg);   color: var(--color-error);   border: 1px solid var(--color-error); }
.li-flash--warning { background: var(--color-warning-bg); color: var(--color-warning); border: 1px solid var(--color-warning); }
```

### Modal Pattern

Modals use the HTML `<dialog>` element. Alpine.js is used only for open/close toggle if needed; HTMX loads modal content on demand.

```html
<!-- Trigger -->
<button class="li-btn li-btn--secondary"
        hx-get="/invoices/{{.ID}}/payment/new"
        hx-target="#modal-container"
        hx-swap="innerHTML"
        onclick="document.getElementById('app-modal').showModal()">
  Record Payment
</button>

<!-- Container always in base layout -->
<dialog id="app-modal">
  <div id="modal-container">
    <!-- HTMX loads content here -->
  </div>
</dialog>
```

---

## PDF Template Design (InvoiceTemplate.md — canonical)

The PDF output is a Go struct rendered through `internal/pdf/pdf.go` using `go-pdf/fpdf`. The visual specification follows **InvoiceTemplate.md** exactly:

### PDF Color Tokens (fpdf RGB values)

```go
var (
  ColorTextPrimary   = fpdf.RGBColorFromHex("#333333")
  ColorTextSecondary = fpdf.RGBColorFromHex("#777777")
  ColorSurfaceAlt    = fpdf.RGBColorFromHex("#F9F9F9")
  ColorBorder        = fpdf.RGBColorFromHex("#EAEAEA")
  ColorBackground    = fpdf.RGBColorFromHex("#FFFFFF")
)
```

### PDF Layout (A4 — 210×297mm, 0.75in margins)

**Header (top 15%)**
- Left: Company logo (max 40mm wide, auto-height) + Company name (16pt Bold, #333333) + Address (9pt Regular, #777777) + Contact (9pt Regular, #777777)
- Right: "INVOICE" (24pt Bold, #333333, wide letter-spacing) + mini-grid: Invoice No / Date / Due Date (labels 10pt Regular #777777, values 10pt SemiBold #333333)

**Billing Section (below header)**
- Left: "BILL TO:" (10pt Bold #777777 Uppercase) + Client Name (12pt SemiBold #333333) + Client details (10pt Regular #777777)
- Right: "SHIP TO:" or "PROJECT:" same style

**Line Items Table (middle, 100% width)**
- Header row background: #F9F9F9, text: 9pt SemiBold #333333 Uppercase
- Columns: Description (50%) | Price (15%) | Qty (15%) | Total (20%)
- Row borders: 1px solid #EAEAEA
- Item name: 10pt SemiBold #333333
- Item description: 9pt Regular #777777
- Numbers: 10pt Regular #333333

**For 3D Print line items, PDF shows the cost breakdown in a collapsible sub-row:**
- Gray sub-row: Filament: Xg @ Rs Y/g (profit Z%) = Rs A | Time: Xh @ Rs Y/h = Rs B | Labour: Rs C | etc.

**Totals (lower right, 50% width)**
- Subtotal / Tax / Discount rows: 10pt Regular #333333
- Divider: 2px solid #333333 (full width of totals block)
- "TOTAL DUE:" 12pt Bold #333333 | Amount: 14pt Bold #333333

**Footer (pinned to bottom)**
- Left: "Payment Terms & Methods" (10pt SemiBold #333333) + bank/payment details (9pt Regular #777777)
- Right: "Thank you for your business!" (10pt SemiBold #333333)
- Optional watermark: centered diagonal text, 36pt, #EAEAEA, rotated 45°

---

## Responsive Breakpoints

The app targets desktop-first since it's a business tool used on a desktop browser. Mobile is supported but not the primary target.

```css
/* Desktop (default): 1024px+ */
/* Tablet: 768px–1023px — sidebar collapses to icon strip */
/* Mobile: <768px — sidebar becomes hamburger menu */

@media (max-width: 1023px) {
  .li-sidebar { width: 60px; }
  .li-sidebar .li-nav-label { display: none; }
}
@media (max-width: 767px) {
  .li-sidebar { position: fixed; transform: translateX(-100%); }
  .li-sidebar.open { transform: translateX(0); }
  .li-main { margin-left: 0; }
}
```

---

## HTMX Loading Indicators

```html
<!-- Global spinner in top nav -->
<span id="global-spinner" class="li-spinner htmx-indicator" aria-hidden="true"></span>
```

```css
.li-spinner {
  display: none;
  width: 16px; height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-text-primary);
  border-radius: 50%;
  animation: li-spin 0.6s linear infinite;
}
.htmx-request .li-spinner,
.htmx-request.li-spinner { display: inline-block; }
@keyframes li-spin { to { transform: rotate(360deg); } }
```

---

## PicoCSS Override Strategy

PicoCSS uses CSS custom properties prefixed with `--pico-`. LayerInvoice overrides them in `:root` within `app.css`:

```css
:root {
  --pico-font-family: 'Noto Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --pico-font-size: 0.875rem;
  --pico-background-color: var(--color-background);
  --pico-color: var(--color-text-primary);
  --pico-secondary-color: var(--color-text-secondary);
  --pico-border-radius: 6px;
  --pico-border-color: var(--color-border);
  --pico-primary: var(--color-accent);
  --pico-primary-hover: var(--color-accent-hover);
  --pico-primary-foreground: var(--color-text-inverse);
  --pico-form-element-background-color: var(--color-surface-alt);
  --pico-form-element-border-color: var(--color-border);
  --pico-table-border-color: var(--color-border);
  --pico-table-row-stripped-background-color: var(--color-surface-alt);
}
```
