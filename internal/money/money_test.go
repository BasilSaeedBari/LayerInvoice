package money

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoneySymbol(t *testing.T) {
	assert.Equal(t, "Rs", New(100, "PKR").Symbol())
	assert.Equal(t, "$", New(100, "USD").Symbol())
	assert.Equal(t, "€", New(100, "EUR").Symbol())
	assert.Equal(t, "£", New(100, "GBP").Symbol())
	assert.Equal(t, "CAD", New(100, "CAD").Symbol())
}

func TestMoneyString(t *testing.T) {
	assert.Equal(t, "Rs 1,250.50", New(125050, "PKR").String())
	assert.Equal(t, "$ -5.00", New(-500, "USD").String())
	assert.Equal(t, "Rs 0.00", Zero("PKR").String())
}

func TestFormatAmount(t *testing.T) {
	assert.Equal(t, "1,000,000.00", FormatAmount(100000000))
	assert.Equal(t, "123,456.78", FormatAmount(12345678))
	assert.Equal(t, "0.05", FormatAmount(5))
	assert.Equal(t, "-12.50", FormatAmount(-1250))
}

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		currency string
		expected int64
		hasError bool
	}{
		{"1250.50", "PKR", 125050, false},
		{"Rs 1,250.50", "PKR", 125050, false},
		{"$1,250.50", "USD", 125050, false},
		{"-12.50", "USD", -1250, false},
		{"1250", "PKR", 125000, false},
		{"0.05", "PKR", 5, false},
		{".05", "PKR", 5, false},
		{"", "PKR", 0, false},
		{"  ", "PKR", 0, false},
		{"12.345", "PKR", 1235, false}, // Rounds up
		{"12.344", "PKR", 1234, false}, // Rounds down
		{"12..34", "PKR", 0, true},     // Invalid decimal
		{"abc", "PKR", 0, false},       // Clean removes all, results in zero
	}

	for _, tt := range tests {
		m, err := Parse(tt.input, tt.currency)
		if tt.hasError {
			assert.Error(t, err, "expected error for: %s", tt.input)
		} else {
			assert.NoError(t, err, "unexpected error for: %s", tt.input)
			assert.Equal(t, tt.expected, m.Amount)
			assert.Equal(t, tt.currency, m.Currency)
		}
	}
}
