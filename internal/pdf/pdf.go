package pdf

import (
	"bytes"
	"fmt"
	"layerinvoice/internal/money"

	"github.com/go-pdf/fpdf"
)

// PDFDocument models standard visual variables.
type PDFDocument struct {
	Title           string // "INVOICE" or "QUOTE"
	DocNumber       string // "INV-0001" or "QTE-0001"
	IssueDate       string
	ExpiryOrDueDate string // Due Date / Expiry Date
	Currency        string // PKR / USD

	SellerName    string
	SellerAddress string
	SellerContact string // Phone | Email

	BuyerName    string
	BuyerCompany string
	BuyerAddress string
	BuyerContact string // Email | Phone

	LineItems []PDFLineItem

	Subtotal      int64
	TaxRate       int64
	TaxAmount     int64
	DiscountValue int64
	Total         int64

	Notes string
	Terms string
}

// PDFLineItem structures raw quantities and descriptions.
type PDFLineItem struct {
	Name        string
	Description string
	Quantity    string
	Unit        string
	UnitPrice   int64
	LineTotal   int64
}

// Generate renders the A4 layout using the go-pdf/fpdf engine and returns raw PDF bytes.
func Generate(doc PDFDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(19.05, 19.05, 19.05) // 0.75 inch margins
	pdf.AddPage()

	// Color definitions
	colorTextPrimary := []int{51, 51, 51}    // #333333
	colorTextSecondary := []int{119, 119, 119} // #777777
	colorSurfaceAlt := []int{249, 249, 249}    // #F9F9F9

	// --- 1. HEADER SECTION ---
	// Left Column: Seller/Company info
	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(85, 6, doc.SellerName, "", 0, "L", false, 0, "")
	
	// Right Column: Doc Title
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(85, 6, doc.Title, "", 1, "R", false, 0, "")

	// Seller sub-details
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
	
	// Alignments
	x := pdf.GetX()
	y := pdf.GetY()

	// Left sub-details (multiple lines)
	pdf.SetXY(x, y+2)
	pdf.MultiCell(85, 4, doc.SellerAddress+"\n"+doc.SellerContact, "", "L", false)
	
	// Right sub-details: metadata grid
	pdf.SetXY(x+90, y+2)
	pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
	pdf.SetFont("Helvetica", "", 10)
	
	metaLabels := []string{doc.Title + " No:", "Date:", "Due Date:"}
	if doc.Title == "QUOTE" {
		metaLabels[2] = "Expiry Date:"
	}
	metaValues := []string{doc.DocNumber, doc.IssueDate, doc.ExpiryOrDueDate}

	for i := 0; i < 3; i++ {
		pdf.SetX(x + 95)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
		pdf.CellFormat(35, 5, metaLabels[i], "", 0, "R", false, 0, "")
		
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
		pdf.CellFormat(40, 5, metaValues[i], "", 1, "R", false, 0, "")
	}

	pdf.Ln(15)

	// --- 2. BILL TO SECTION ---
	y = pdf.GetY()
	pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(85, 4, "BILL TO:", "", 0, "L", false, 0, "")
	
	// We can leave right side blank or projects info
	pdf.CellFormat(85, 4, "PAYMENT CURRENCY:", "", 1, "R", false, 0, "")

	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(85, 6, doc.BuyerName, "", 0, "L", false, 0, "")
	
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(85, 6, doc.Currency, "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
	
	buyerInfo := doc.BuyerCompany
	if doc.BuyerAddress != "" {
		if buyerInfo != "" {
			buyerInfo += "\n"
		}
		buyerInfo += doc.BuyerAddress
	}
	if doc.BuyerContact != "" {
		if buyerInfo != "" {
			buyerInfo += "\n"
		}
		buyerInfo += doc.BuyerContact
	}
	
	pdf.SetX(19.05)
	pdf.MultiCell(85, 4.5, buyerInfo, "", "L", false)

	pdf.Ln(10)

	// --- 3. ITEMS TABLE ---
	// Headers
	pdf.SetFillColor(colorSurfaceAlt[0], colorSurfaceAlt[1], colorSurfaceAlt[2])
	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
	pdf.SetFont("Helvetica", "B", 9)

	pdf.CellFormat(90, 8, "ITEM DESCRIPTION", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "PRICE", "1", 0, "R", true, 0, "")
	pdf.CellFormat(20, 8, "QTY", "1", 0, "C", true, 0, "")
	pdf.CellFormat(36, 8, "TOTAL", "1", 1, "R", true, 0, "")

	// Rows
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])

	symbol := "Rs"
	if doc.Currency == "USD" {
		symbol = "$"
	} else if doc.Currency == "EUR" {
		symbol = "€"
	} else if doc.Currency == "GBP" {
		symbol = "£"
	}

	for _, item := range doc.LineItems {
		// Calculate dynamic heights if name or description spans multiple lines
		descText := item.Description
		
		yStart := pdf.GetY()
		
		// Render description cell (left-aligned)
		pdf.SetXY(19.05, yStart+1)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(90, 4, item.Name, "", 1, "L", false, 0, "")
		
		if descText != "" {
			pdf.SetX(19.05)
			pdf.SetFont("Helvetica", "I", 8)
			pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
			pdf.MultiCell(90, 3.5, descText, "", "L", false)
		}
		
		yEnd := pdf.GetY()
		rowHeight := (yEnd - yStart) + 2
		if rowHeight < 8 {
			rowHeight = 8
		}

		// Draw border and numbers
		pdf.SetXY(19.05, yStart)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
		
		// Description container border cell
		pdf.CellFormat(90, rowHeight, "", "B", 0, "L", false, 0, "")
		
		// Prices cells
		pdf.CellFormat(25, rowHeight, fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(item.UnitPrice)), "B", 0, "R", false, 0, "")
		pdf.CellFormat(20, rowHeight, item.Quantity+" "+item.Unit, "B", 0, "C", false, 0, "")
		pdf.CellFormat(36, rowHeight, fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(item.LineTotal)), "B", 1, "R", false, 0, "")
	}

	pdf.Ln(6)

	// --- 4. TOTALS BLOCK ---
	y = pdf.GetY()
	
	// Left side notes/terms
	pdf.SetXY(19.05, y)
	pdf.SetTextColor(colorTextSecondary[0], colorTextSecondary[1], colorTextSecondary[2])
	
	if doc.Terms != "" {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(100, 4, "Terms & Conditions:", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(100, 3.5, doc.Terms, "", "L", false)
		pdf.Ln(4)
	}
	
	if doc.Notes != "" {
		pdf.SetX(19.05)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(100, 4, "Notes & Payment Instructions:", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(100, 3.5, doc.Notes, "", "L", false)
	}

	// Right side totals
	pdf.SetXY(125, y)
	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])

	// Subtotal
	pdf.SetX(125)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(25, 5.5, "Subtotal:", "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5.5, fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(doc.Subtotal)), "", 1, "R", false, 0, "")

	// Discount
	if doc.DiscountValue > 0 {
		pdf.SetX(125)
		pdf.CellFormat(25, 5.5, "Discount:", "", 0, "L", false, 0, "")
		pdf.CellFormat(40, 5.5, fmt.Sprintf("-%s %s", symbol, FormatMoneyRaw(doc.DiscountValue)), "", 1, "R", false, 0, "")
	}

	// Tax
	pdf.SetX(125)
	taxPct := float64(doc.TaxRate) / 100.0
	pdf.CellFormat(25, 5.5, fmt.Sprintf("Tax (%.2f%%):", taxPct), "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 5.5, fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(doc.TaxAmount)), "", 1, "R", false, 0, "")

	// Divider
	pdf.SetX(125)
	pdf.SetLineWidth(0.5)
	pdf.SetDrawColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
	pdf.CellFormat(65, 2, "", "T", 1, "C", false, 0, "")

	// Grand Total
	pdf.SetX(125)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(25, 6, "TOTAL DUE:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(40, 6, fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(doc.Total)), "", 1, "R", false, 0, "")

	// --- 5. FOOTER ---
	// Pinned gratitude
	pdf.SetY(265)
	pdf.SetTextColor(colorTextPrimary[0], colorTextPrimary[1], colorTextPrimary[2])
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(171.9, 5, "Thank you for your business!", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF output buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// FormatMoneyRaw is a tiny internal helper to write format logic without package cycles.
func FormatMoneyRaw(amount int64) string {
	return money.FormatAmount(amount)
}
