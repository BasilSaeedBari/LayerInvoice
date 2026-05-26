# Invoice Template Design Specification

## 1. Global Styles & Assets

**Typography**
* **Primary Font:** Noto Sans (Google Fonts)
* **URL:** `https://fonts.google.com/specimen/Noto+Sans`
* **Weights Used:** Regular (400), Semi-Bold (600), Bold (700)
* **Base Font Size:** 10pt for standard text, 12pt for subheadings, 24pt for main titles.

**Color Palette**
* **Primary Text (Dark):** `#333333` (Used for major headings, totals, and primary data)
* **Secondary Text (Gray):** `#777777` (Used for labels, contact info, and terms)
* **Accent Background:** `#F9F9F9` (Used for the table header row)
* **Borders & Dividers:** `#EAEAEA` (Used for table row dividers and section separations)
* **Background:** `#FFFFFF` (Pure white)

**Document Layout**
* **Size:** Standard A4 (210mm x 297mm) or US Letter (8.5" x 11")
* **Margins:** 0.75 inches (top, bottom, left, right)
* **Grid System:** 12-column layout principle, divided mostly into 50/50 splits for the header/billing sections, and a 100% width block for the table.

---

## 2. Component Breakdown

### A. Header Section (Top)
This section occupies the top 15% of the document and is split into two equal columns.

**Left Column (Company Details)**
* **Alignment:** Left-aligned
* **Company Logo / Name:** * Text: "YOUR LOGO" or Company Name
    * Style: Noto Sans Bold, 16pt, `#333333`
* **Company Address Block:**
    * Text: Address Line 1, City, State, ZIP
    * Style: Noto Sans Regular, 9pt, `#777777`
    * Spacing: 1.2 line height.
* **Contact Info:**
    * Text: Phone | Email | Website
    * Style: Noto Sans Regular, 9pt, `#777777`

**Right Column (Invoice Details)**
* **Alignment:** Right-aligned
* **Document Title:** * Text: "INVOICE"
    * Style: Noto Sans Bold, 24pt, `#333333`, uppercase, tracking (letter-spacing) set to slightly wide.
* **Invoice Meta Data (Grid format):**
    * Create a mini 2-column layout aligned to the right edge.
    * **Labels (Left side of mini-grid):** "Invoice No:", "Date:", "Due Date:" (Noto Sans Regular, 10pt, `#777777`)
    * **Values (Right side of mini-grid):** "#INV-001", "DD/MM/YYYY", "DD/MM/YYYY" (Noto Sans Semi-Bold, 10pt, `#333333`)

---

### B. Billing Information Section (Upper Middle)
Positioned roughly 2 inches from the top, separated from the header by whitespace (no dividing line).

**Left Column (Bill To)**
* **Header:** "BILL TO:"
    * Style: Noto Sans Bold, 10pt, `#777777`, Uppercase.
    * Margin-bottom: 0.25 inches.
* **Client Name:** Noto Sans Semi-Bold, 12pt, `#333333`
* **Client Details:** Address, Phone, Email (Noto Sans Regular, 10pt, `#777777`, 1.5 line height)

**Right Column (Ship To / Project Details - Optional)**
* **Header:** "SHIP TO:" (or "PROJECT:")
    * Style: Noto Sans Bold, 10pt, `#777777`, Uppercase.
    * Margin-bottom: 0.25 inches.
* **Details:** Follows the exact typography and spacing as the "Bill To" block.

---

### C. Itemized Table Section (Middle)
This is a full-width block (100%) starting roughly 4 inches from the top. 

**Table Header Row**
* **Background Color:** `#F9F9F9`
* **Padding:** 10px vertical, 15px horizontal.
* **Text Style:** Noto Sans Semi-Bold, 9pt, `#333333`, Uppercase.
* **Columns & Widths:**
    1.  **Item Description:** Left aligned (approx 50% width)
    2.  **Price:** Right aligned (approx 15% width)
    3.  **Qty:** Center aligned (approx 15% width)
    4.  **Total:** Right aligned (approx 20% width)

**Table Body Rows**
* **Padding:** 12px vertical, 15px horizontal per cell.
* **Border:** A 1px solid bottom border (`#EAEAEA`) on every row to separate items.
* **Text Style (Item Name):** Noto Sans Semi-Bold, 10pt, `#333333`. 
* **Text Style (Item Description beneath name):** Noto Sans Regular, 9pt, `#777777`.
* **Text Style (Numbers/Prices):** Noto Sans Regular, 10pt, `#333333`.

---

### D. Financial Totals Section (Lower Right)
Located immediately underneath the table, pushed entirely to the right half of the page.

* **Structure:** A 2-column mini-grid.
* **Subtotal Row:** Label "Subtotal:" (Regular) | Value "$0.00" (Regular)
* **Tax Row:** Label "Tax (XX%):" (Regular) | Value "$0.00" (Regular)
* **Discount Row:** Label "Discount:" (Regular) | Value "-$0.00" (Regular)
* *(Divider: A 2px solid line `#333333` spanning only the width of this totals block)*
* **Grand Total Row:** * Label: "TOTAL DUE:" (Noto Sans Bold, 12pt, `#333333`)
    * Value: "$0.00" (Noto Sans Bold, 14pt, `#333333`)

---

### E. Footer Section (Bottom)
Located at the absolute bottom of the page, anchored to the bottom margin.

**Left-Aligned (Terms and Payment Info)**
* **Header:** "Payment Terms & Methods"
    * Style: Noto Sans Semi-Bold, 10pt, `#333333`
* **Body Text:** Bank transfer details, PayPal addresses, or late fee terms. 
    * Style: Noto Sans Regular, 9pt, `#777777`, max-width of 60% of the page to prevent text from running into the bottom-right corner.

**Center or Right-Aligned (Gratitude)**
* **Text:** "Thank you for your business!"
* **Style:** Noto Sans Semi-Bold, 10pt, `#333333`.