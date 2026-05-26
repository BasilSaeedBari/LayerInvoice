package money

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Money represents a monetary value in its smallest unit (e.g. Paisa for PKR, Cents for USD).
type Money struct {
	Amount   int64  `json:"amount"`   // Value in smallest unit (e.g. 125050 = Rs 1250.50)
	Currency string `json:"currency"` // PKR, USD, EUR, etc.
}

// New creates a new Money instance.
func New(amount int64, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

// Zero returns a zero Money instance for the given currency.
func Zero(currency string) Money {
	return New(0, currency)
}

// Symbol returns the currency symbol or falls back to the currency code.
func (m Money) Symbol() string {
	switch m.Currency {
	case "PKR":
		return "Rs"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	default:
		return m.Currency
	}
}

// String returns a formatted representation, e.g. "Rs 1,250.50" or "$1,250.50".
func (m Money) String() string {
	symbol := m.Symbol()
	formattedVal := FormatAmount(m.Amount)
	return fmt.Sprintf("%s %s", symbol, formattedVal)
}

// FormatAmount formats raw int64 amount to comma-separated decimal string, e.g. 125050 -> "1,250.50".
func FormatAmount(amount int64) string {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

	cents := amount % 100
	dollars := amount / 100

	dollarsStr := strconv.FormatInt(dollars, 10)
	var sb strings.Builder

	// Insert commas every 3 digits
	n := len(dollarsStr)
	for i, char := range dollarsStr {
		sb.WriteRune(char)
		if (n-i-1)%3 == 0 && i != n-1 {
			sb.WriteRune(',')
		}
	}

	formatted := fmt.Sprintf("%s.%02d", sb.String(), cents)
	if isNegative {
		return "-" + formatted
	}
	return formatted
}

var cleanRegex = regexp.MustCompile(`[^\d\.\-]`)

// Parse converts a user-input decimal string (e.g. "Rs 1,250.50", "1250.50", "1250") into a Money struct.
func Parse(input string, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Money{}, errors.New("currency cannot be empty")
	}

	// Remove currency symbols, spaces, commas
	cleaned := cleanRegex.ReplaceAllString(input, "")
	if cleaned == "" {
		return Zero(currency), nil
	}

	// Find the decimal point
	parts := strings.Split(cleaned, ".")
	if len(parts) > 2 {
		return Money{}, fmt.Errorf("invalid money format: %s", input)
	}

	var rawAmount int64
	if len(parts) == 1 {
		// Whole number, e.g., "1250" -> 125000 cents
		val, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("invalid integer: %s", parts[0])
		}
		rawAmount = val * 100
	} else {
		// Decimal number, e.g., "1250.5" -> 125050 cents
		wholePart := parts[0]
		decimalPart := parts[1]

		if len(decimalPart) == 0 {
			decimalPart = "00"
		} else if len(decimalPart) == 1 {
			decimalPart = decimalPart + "0"
		} else if len(decimalPart) > 2 {
			// Round to nearest cent/paisa (standard ROUND_HALF_UP)
			// Let's take the first 3 chars
			thirdDigit := decimalPart[2]
			decimalPart = decimalPart[:2]
			val, _ := strconv.ParseInt(decimalPart, 10, 64)
			if thirdDigit >= '5' {
				val++
			}
			decimalPart = strconv.FormatInt(val, 10)
		}

		if wholePart == "" || wholePart == "-" {
			wholePart = wholePart + "0"
		}

		valWhole, err := strconv.ParseInt(wholePart, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("invalid whole part: %s", wholePart)
		}

		valDecimal, err := strconv.ParseInt(decimalPart, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("invalid decimal part: %s", decimalPart)
		}

		if valWhole < 0 || strings.HasPrefix(wholePart, "-") {
			rawAmount = valWhole*100 - valDecimal
		} else {
			rawAmount = valWhole*100 + valDecimal
		}
	}

	return New(rawAmount, currency), nil
}
