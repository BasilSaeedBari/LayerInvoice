package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"layerinvoice/internal/money"
	"os"
	"strings"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/phpdave11/gofpdf"
)

// PDFDocument models standard visual variables.
type PDFDocument struct {
	Title           string // "INVOICE" or "QUOTE"
	DocNumber       string // "INV-0001" or "QTE-0001"
	IssueDate       string
	ExpiryOrDueDate string // Due Date / Expiry Date
	Currency        string // PKR / USD
	CompanyLogo     string // Base64 company logo data URL

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

// FormatMoney formats values to regional currencies.
func FormatMoney(amount int64, currency string) string {
	symbol := "Rs"
	if currency == "USD" {
		symbol = "$"
	} else if currency == "EUR" {
		symbol = "€"
	} else if currency == "GBP" {
		symbol = "£"
	}
	return fmt.Sprintf("%s %s", symbol, FormatMoneyRaw(amount))
}

// FormatMoneyRaw formats numeric values back to standard decimal representations.
func FormatMoneyRaw(amount int64) string {
	return money.FormatAmount(amount)
}

// drawDecorations draws the geometric corners and side accent shapes programmatically as vectors.
func drawDecorations(pdf *gofpdf.Fpdf) {
	// Color set
	pdf.SetFillColor(124, 92, 191) // #7c5cbf
	pdf.SetDrawColor(124, 92, 191)

	// --- 1. Top-Left Corner ---
	// L-Shape
	pdf.Polygon([]gofpdf.PointType{
		{X: 15, Y: 15},
		{X: 35, Y: 15},
		{X: 35, Y: 17},
		{X: 17, Y: 17},
		{X: 17, Y: 35},
		{X: 15, Y: 35},
	}, "F")
	// Diamond
	pdf.SetLineWidth(0.4)
	pdf.Polygon([]gofpdf.PointType{
		{X: 25, Y: 20},
		{X: 30, Y: 25},
		{X: 25, Y: 30},
		{X: 20, Y: 25},
	}, "D")
	// Triangles
	pdf.Polygon([]gofpdf.PointType{
		{X: 15, Y: 45},
		{X: 19, Y: 41},
		{X: 19, Y: 49},
	}, "F")
	pdf.Polygon([]gofpdf.PointType{
		{X: 45, Y: 15},
		{X: 41, Y: 19},
		{X: 49, Y: 19},
	}, "F")
	// Dot
	pdf.Circle(32, 32, 0.8, "F")

	// --- 2. Top-Right Corner ---
	// L-Shape
	pdf.Polygon([]gofpdf.PointType{
		{X: 195, Y: 15},
		{X: 175, Y: 15},
		{X: 175, Y: 17},
		{X: 193, Y: 17},
		{X: 193, Y: 35},
		{X: 195, Y: 35},
	}, "F")
	// Diamond
	pdf.Polygon([]gofpdf.PointType{
		{X: 185, Y: 20},
		{X: 180, Y: 25},
		{X: 185, Y: 30},
		{X: 190, Y: 25},
	}, "D")
	// Triangles
	pdf.Polygon([]gofpdf.PointType{
		{X: 195, Y: 45},
		{X: 191, Y: 41},
		{X: 191, Y: 49},
	}, "F")
	pdf.Polygon([]gofpdf.PointType{
		{X: 165, Y: 15},
		{X: 169, Y: 19},
		{X: 161, Y: 19},
	}, "F")
	// Dot
	pdf.Circle(178, 32, 0.8, "F")

	// --- 3. Bottom-Left Corner ---
	// L-Shape
	pdf.Polygon([]gofpdf.PointType{
		{X: 15, Y: 282},
		{X: 35, Y: 282},
		{X: 35, Y: 280},
		{X: 17, Y: 280},
		{X: 17, Y: 262},
		{X: 15, Y: 262},
	}, "F")
	// Diamond
	pdf.Polygon([]gofpdf.PointType{
		{X: 25, Y: 277},
		{X: 30, Y: 272},
		{X: 25, Y: 267},
		{X: 20, Y: 272},
	}, "D")
	// Triangles
	pdf.Polygon([]gofpdf.PointType{
		{X: 15, Y: 252},
		{X: 19, Y: 256},
		{X: 19, Y: 248},
	}, "F")
	pdf.Polygon([]gofpdf.PointType{
		{X: 45, Y: 282},
		{X: 41, Y: 278},
		{X: 49, Y: 278},
	}, "F")
	// Dot
	pdf.Circle(32, 265, 0.8, "F")

	// --- 4. Bottom-Right Corner ---
	// L-Shape
	pdf.Polygon([]gofpdf.PointType{
		{X: 195, Y: 282},
		{X: 175, Y: 282},
		{X: 175, Y: 280},
		{X: 193, Y: 280},
		{X: 193, Y: 262},
		{X: 195, Y: 262},
	}, "F")
	// Diamond
	pdf.Polygon([]gofpdf.PointType{
		{X: 185, Y: 277},
		{X: 180, Y: 272},
		{X: 185, Y: 267},
		{X: 190, Y: 272},
	}, "D")
	// Triangles
	pdf.Polygon([]gofpdf.PointType{
		{X: 195, Y: 252},
		{X: 191, Y: 256},
		{X: 191, Y: 248},
	}, "F")
	pdf.Polygon([]gofpdf.PointType{
		{X: 165, Y: 282},
		{X: 169, Y: 278},
		{X: 161, Y: 278},
	}, "F")
	// Dot
	pdf.Circle(178, 265, 0.8, "F")

	// --- Side Diamond Accents ---
	pdf.Polygon([]gofpdf.PointType{
		{X: 15, Y: 143.5},
		{X: 20, Y: 148.5},
		{X: 15, Y: 153.5},
		{X: 10, Y: 148.5},
	}, "D")
	pdf.Polygon([]gofpdf.PointType{
		{X: 195, Y: 143.5},
		{X: 200, Y: 148.5},
		{X: 195, Y: 153.5},
		{X: 190, Y: 148.5},
	}, "D")
}

// Generate renders the A4 invoice/quote layout using pure-Go gofpdf (releasing all external browser engines).
func Generate(doc PDFDocument, browserPath string) ([]byte, error) {
	// Initialize A4 Portrait Page, measuring in millimeters (mm)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Draw Background Accent Corner Vector Graphics
	drawDecorations(pdf)

	// --- HEADER SECTION (Y=20) ---
	var headerTextY float64 = 52

	// Render Logo
	if doc.CompanyLogo != "" && strings.HasPrefix(doc.CompanyLogo, "data:image/") {
		commaIdx := strings.Index(doc.CompanyLogo, ",")
		if commaIdx != -1 {
			base64Data := doc.CompanyLogo[commaIdx+1:]
			decBytes, err := base64.StdEncoding.DecodeString(base64Data)
			if err == nil {
				imgType := "PNG"
				if strings.Contains(doc.CompanyLogo[:commaIdx], "image/jpeg") || strings.Contains(doc.CompanyLogo[:commaIdx], "image/jpg") {
					imgType = "JPG"
				} else if strings.Contains(doc.CompanyLogo[:commaIdx], "image/gif") {
					imgType = "GIF"
				}

				// Create safe temp logo
				tmpLogo, err := os.CreateTemp("", "logo_*." + strings.ToLower(imgType))
				if err == nil {
					_, _ = tmpLogo.Write(decBytes)
					tmpLogo.Close()
					defer os.Remove(tmpLogo.Name())

					// Register logo in FPDF
					pdf.RegisterImage(tmpLogo.Name(), imgType)

					// Dynamically check aspect ratio to fit beautifully within a 40x25mm centered bounding box
					var imgW, imgH float64
					maxW := 40.0
					maxH := 25.0
					
					reader := bytes.NewReader(decBytes)
					imgConfig, _, err := image.DecodeConfig(reader)
					if err == nil && imgConfig.Width > 0 && imgConfig.Height > 0 {
						aspect := float64(imgConfig.Width) / float64(imgConfig.Height)
						if aspect > maxW/maxH {
							// Width constrained
							imgW = maxW
							imgH = maxW / aspect
						} else {
							// Height constrained
							imgH = maxH
							imgW = maxH * aspect
						}
					} else {
						// Default fallback size if we cannot read config
						imgW = 25.0
						imgH = 25.0
					}

					// Center horizontally and vertically within our 40x25mm header box
					imgX := 105.0 - (imgW / 2.0)
					imgY := 20.0 + (maxH - imgH) / 2.0

					pdf.Image(tmpLogo.Name(), imgX, imgY, imgW, imgH, false, imgType, 0, "")
					headerTextY = 54
				}
			}
		}
	} else {
		// Fallback Logo: Intersecting Ellipses
		// Ellipse 1 (fill #4a3a6b)
		pdf.SetFillColor(74, 58, 107)
		pdf.Ellipse(95, 27, 7, 6, 0, "F")
		// Ellipse 2 (fill #7c5cbf)
		pdf.SetFillColor(124, 92, 191)
		pdf.Ellipse(103, 27, 7, 6, 0, "F")
		headerTextY = 54
	}

	// Centered Company Name
	pdf.SetTextColor(124, 92, 191) // #7c5cbf
	pdf.SetFont("Arial", "B", 10)
	companyName := strings.ToUpper(doc.SellerName)
	spacedName := ""
	for _, ch := range companyName {
		spacedName += string(ch) + "  "
	}
	spacedName = strings.TrimSpace(spacedName)
	pdf.SetXY(15, headerTextY)
	pdf.CellFormat(180, 5, spacedName, "", 0, "C", false, 0, "")

	// Document Title
	pdf.SetTextColor(74, 58, 107) // #4a3a6b
	pdf.SetFont("Arial", "B", 36)
	pdf.SetXY(15, headerTextY+7)
	pdf.CellFormat(180, 12, doc.Title, "", 0, "C", false, 0, "")

	// Document Number & Dates
	pdf.SetTextColor(102, 102, 102) // #666
	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(15, headerTextY+21)
	pdf.CellFormat(180, 5, "N. "+doc.DocNumber, "", 0, "C", false, 0, "")

	dateStr := "Date: " + doc.IssueDate
	if doc.ExpiryOrDueDate != "" {
		if doc.Title == "QUOTE" {
			dateStr += "  |  Valid Until: " + doc.ExpiryOrDueDate
		} else {
			dateStr += "  |  Due Date: " + doc.ExpiryOrDueDate
		}
	}
	pdf.SetXY(15, headerTextY+26)
	pdf.CellFormat(180, 5, dateStr, "", 0, "C", false, 0, "")

	// --- BILLING LEDGER COLUMNS (Three side-by-side grids) ---
	ledgerY := headerTextY + 36
	pdf.SetDrawColor(232, 224, 240) // #e8e0f0
	pdf.SetLineWidth(0.3)
	pdf.Line(15, ledgerY, 195, ledgerY)
	pdf.Line(15, ledgerY+32, 195, ledgerY+32)

	// Column 1: Bill From
	pdf.SetXY(15, ledgerY+3)
	pdf.SetTextColor(124, 92, 191)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(55, 4, "BILL FROM", "", 0, "L", false, 0, "")

	pdf.SetXY(15, ledgerY+8)
	pdf.SetTextColor(51, 51, 51)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(55, 4, doc.SellerName, "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(68, 68, 68)
	curY := ledgerY + 13
	sellerAddrLines := strings.Split(doc.SellerAddress, "\n")
	for _, l := range sellerAddrLines {
		line := strings.TrimSpace(l)
		if line != "" {
			pdf.SetXY(15, curY)
			pdf.CellFormat(55, 3.8, line, "", 0, "L", false, 0, "")
			curY += 3.8
		}
	}
	if doc.SellerContact != "" {
		pdf.SetXY(15, curY)
		pdf.CellFormat(55, 3.8, doc.SellerContact, "", 0, "L", false, 0, "")
	}

	// Column 2: Bill To
	pdf.SetXY(75, ledgerY+3)
	pdf.SetTextColor(124, 92, 191)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(55, 4, "BILL TO", "", 0, "L", false, 0, "")

	pdf.SetXY(75, ledgerY+8)
	pdf.SetTextColor(51, 51, 51)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(55, 4, doc.BuyerName, "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(68, 68, 68)
	curY = ledgerY + 13
	if doc.BuyerCompany != "" {
		pdf.SetXY(75, curY)
		pdf.CellFormat(55, 3.8, doc.BuyerCompany, "", 0, "L", false, 0, "")
		curY += 3.8
	}
	buyerAddrLines := strings.Split(doc.BuyerAddress, "\n")
	for _, l := range buyerAddrLines {
		line := strings.TrimSpace(l)
		if line != "" {
			pdf.SetXY(75, curY)
			pdf.CellFormat(55, 3.8, line, "", 0, "L", false, 0, "")
			curY += 3.8
		}
	}
	if doc.BuyerContact != "" {
		pdf.SetXY(75, curY)
		pdf.CellFormat(55, 3.8, doc.BuyerContact, "", 0, "L", false, 0, "")
	}

	// Column 3: Payment Details
	pdf.SetXY(135, ledgerY+3)
	pdf.SetTextColor(124, 92, 191)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(60, 4, "PAYMENT / DETAILS", "", 0, "L", false, 0, "")

	pdf.SetXY(135, ledgerY+8)
	pdf.SetTextColor(102, 102, 102)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(60, 4, "BILLED CURRENCY", "", 0, "L", false, 0, "")

	pdf.SetXY(135, ledgerY+12)
	pdf.SetTextColor(51, 51, 51)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 4, doc.Currency, "", 0, "L", false, 0, "")

	if doc.Notes != "" {
		pdf.SetXY(135, ledgerY+17)
		pdf.SetTextColor(102, 102, 102)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(60, 4, "INSTRUCTIONS", "", 0, "L", false, 0, "")

		pdf.SetXY(135, ledgerY+21)
		pdf.SetTextColor(68, 68, 68)
		pdf.SetFont("Arial", "", 8)
		// Wrapped Instructions
		pdf.MultiCell(60, 3.2, doc.Notes, "", "L", false)
	}

	// --- TABLE GAUGE (Y=headerTextY+74) ---
	tableY := ledgerY + 38
	pdf.SetFillColor(124, 92, 191) // #7c5cbf
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8.5)

	pdf.SetXY(15, tableY)
	pdf.CellFormat(15, 8, "N", "", 0, "C", true, 0, "")
	pdf.CellFormat(95, 8, "DESCRIPTION", "", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "PRICE", "", 0, "R", true, 0, "")
	pdf.CellFormat(20, 8, "QTY", "", 0, "R", true, 0, "")
	pdf.CellFormat(25, 8, "TOTAL", "", 0, "R", true, 0, "")

	y := tableY + 8
	pdf.SetLineWidth(0.2)

	for i, item := range doc.LineItems {
		// Clean and replace Unicode bullets "•" or "â€¢" with "-" to fit WinAnsiEncoding
		cleanDesc := strings.ReplaceAll(item.Description, "•", "-")
		cleanDesc = strings.ReplaceAll(cleanDesc, "â€¢", "-")
		cleanDesc = strings.ReplaceAll(cleanDesc, "\u2022", "-")

		// Calculate custom wrapped height based on multi-line sub-description wrapping inside 93mm area
		pdf.SetFont("Arial", "", 8)
		lines := pdf.SplitLines([]byte(cleanDesc), 93)
		
		descHeight := float64(len(lines))*3.6 + 6 // padding + line heights
		if descHeight < 11 {
			descHeight = 11
		}

		// Trigger automatic multi-page overflow breaks cleanly
		if y+descHeight > 235 {
			pdf.AddPage()
			drawDecorations(pdf)
			
			// Redraw table headers on new page
			pdf.SetFillColor(124, 92, 191)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Arial", "B", 8.5)
			pdf.SetXY(15, 20)
			pdf.CellFormat(15, 8, "N", "", 0, "C", true, 0, "")
			pdf.CellFormat(95, 8, "DESCRIPTION", "", 0, "L", true, 0, "")
			pdf.CellFormat(25, 8, "PRICE", "", 0, "R", true, 0, "")
			pdf.CellFormat(20, 8, "QTY", "", 0, "R", true, 0, "")
			pdf.CellFormat(25, 8, "TOTAL", "", 0, "R", true, 0, "")
			
			y = 28
		}

		// Zebra Stripes alternating backgrounds
		if i%2 == 1 {
			pdf.SetFillColor(249, 247, 252) // #f9f7fc
			pdf.Rect(15, y, 180, descHeight, "F")
		}

		// Item Number (Vertically Centered)
		pdf.SetTextColor(124, 92, 191)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.SetXY(15, y + (descHeight-4)/2)
		pdf.CellFormat(15, 4, fmt.Sprintf("%d", i+1), "", 0, "C", false, 0, "")

		// Bold Description Title & wrapped cost breakdown details
		pdf.SetTextColor(74, 58, 107) // #4a3a6b
		pdf.SetFont("Arial", "B", 8.5)
		pdf.SetXY(30, y+2)
		pdf.CellFormat(95, 4, item.Name, "", 0, "L", false, 0, "")

		if cleanDesc != "" {
			pdf.SetTextColor(102, 102, 102) // #666
			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(30, y+6.2)
			pdf.MultiCell(93, 3.2, cleanDesc, "", "L", false)
		}

		// Price Column
		pdf.SetTextColor(68, 68, 68)
		pdf.SetFont("Arial", "", 8.5)
		pdf.SetXY(125, y + (descHeight-4)/2)
		pdf.CellFormat(25, 4, FormatMoney(item.UnitPrice, doc.Currency), "", 0, "R", false, 0, "")

		// Quantity Column
		pdf.SetXY(150, y + (descHeight-4)/2)
		qtyStr := item.Quantity
		if item.Unit != "" && item.Unit != "pcs" {
			qtyStr += " " + item.Unit
		}
		pdf.CellFormat(20, 4, qtyStr, "", 0, "R", false, 0, "")

		// Item Total Column
		pdf.SetTextColor(51, 51, 51)
		pdf.SetFont("Arial", "B", 8.5)
		pdf.SetXY(170, y + (descHeight-4)/2)
		pdf.CellFormat(25, 4, FormatMoney(item.LineTotal, doc.Currency), "", 0, "R", false, 0, "")

		// Draw border line underneath row
		pdf.SetDrawColor(232, 224, 240)
		pdf.Line(15, y+descHeight, 195, y+descHeight)

		y += descHeight
	}

	// --- TOTALS, TERMS, AND NOTES PANEL ---
	if y+45 > 235 {
		pdf.AddPage()
		drawDecorations(pdf)
		y = 20
	}

	totalsY := y + 5

	// Subtotal Row
	pdf.SetTextColor(102, 102, 102)
	pdf.SetFont("Arial", "", 8.5)
	pdf.SetXY(135, totalsY)
	pdf.CellFormat(30, 4, "SUBTOTAL", "", 0, "L", false, 0, "")
	pdf.SetTextColor(68, 68, 68)
	pdf.SetXY(165, totalsY)
	pdf.CellFormat(30, 4, FormatMoney(doc.Subtotal, doc.Currency), "", 0, "R", false, 0, "")
	totalsY += 4.5

	// Discount Row (if > 0)
	if doc.DiscountValue > 0 {
		pdf.SetTextColor(102, 102, 102)
		pdf.SetXY(135, totalsY)
		pdf.CellFormat(30, 4, "DISCOUNT", "", 0, "L", false, 0, "")
		pdf.SetTextColor(68, 68, 68)
		pdf.SetXY(165, totalsY)
		pdf.CellFormat(30, 4, "-" + FormatMoney(doc.DiscountValue, doc.Currency), "", 0, "R", false, 0, "")
		totalsY += 4.5
	}

	// Tax Row (if > 0)
	if doc.TaxAmount > 0 {
		pdf.SetTextColor(102, 102, 102)
		pdf.SetXY(135, totalsY)
		taxRateLabel := fmt.Sprintf("TAX (%.2f%%)", float64(doc.TaxRate)/100.0)
		pdf.CellFormat(30, 4, taxRateLabel, "", 0, "L", false, 0, "")
		pdf.SetTextColor(68, 68, 68)
		pdf.SetXY(165, totalsY)
		pdf.CellFormat(30, 4, FormatMoney(doc.TaxAmount, doc.Currency), "", 0, "R", false, 0, "")
		totalsY += 4.5
	}

	// Total Due Highlight Banner
	pdf.SetFillColor(124, 92, 191) // #7c5cbf
	pdf.Rect(135, totalsY+1.5, 60, 8, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetXY(138, totalsY+3.5)
	pdf.CellFormat(25, 4, "TOTAL DUE", "", 0, "L", false, 0, "")
	pdf.SetXY(163, totalsY+3.5)
	pdf.CellFormat(30, 4, FormatMoney(doc.Total, doc.Currency), "", 0, "R", false, 0, "")

	// Legal Terms & Instructions (Left hand side)
	leftY := y + 5
	if doc.Terms != "" {
		pdf.SetTextColor(124, 92, 191)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(15, leftY)
		pdf.CellFormat(110, 4, "TERMS & CONDITIONS", "", 0, "L", false, 0, "")

		pdf.SetTextColor(102, 102, 102)
		pdf.SetFont("Arial", "", 7.5)
		pdf.SetXY(15, leftY+4.2)
		pdf.MultiCell(110, 3.2, doc.Terms, "", "L", false)
	}

	// --- PINNED STATIC FOOTER (Y=250) ---
	pdf.SetDrawColor(232, 224, 240) // #e8e0f0
	pdf.SetLineWidth(0.3)
	pdf.Line(15, 250, 195, 250)

	pdf.SetTextColor(124, 92, 191)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(15, 253)
	pdf.CellFormat(180, 5, "THANK YOU!", "", 0, "C", false, 0, "")

	pdf.SetTextColor(136, 136, 136)
	pdf.SetFont("Arial", "", 8)
	pdf.SetXY(15, 258)
	pdf.CellFormat(180, 4, "Thank you for your business. LayerInvoice dynamic fabrication services.", "", 0, "C", false, 0, "")

	// Compile into bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to compile gofpdf compilation buffer: %w", err)
	}

	return buf.Bytes(), nil
}
