package calculator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculate(t *testing.T) {
	// Let's create an input scenario:
	// Filament: Rs 5.00/g, using 50.0g
	// Filament markup: 100% (10000 bp)
	// Time: Rs 2.00/hr, using 4.5h
	// Labor: Rs 1.50
	// Electricity: Rs 0.50
	// Postproc: Rs 1.00
	// Packaging: Rs 2.50
	// Shipping: Rs 5.00
	// Failure rate: 5% (500 bp)
	// Profit multiplier: 50% (5000 bp)
	in := PrintJobInputs{
		Grams:             50.0,
		CostPerGram:       500, // Rs 5.00 = 500 Paisa
		FilamentProfitPct: 100, // 100% markup

		PrintHours:      4.5,
		TimeRate:        200, // Rs 2.00 = 200 Paisa
		LabourCost:      150, // Rs 1.50 = 150 Paisa
		ElectricityCost: 50,  // Rs 0.50 = 50 Paisa
		PostprocCost:    100, // Rs 1.00 = 100 Paisa

		PackagingCost: 250, // Rs 2.50 = 250 Paisa
		ShippingCost:  500, // Rs 5.00 = 500 Paisa

		FailureRate:      5,  // 5% failure
		ProfitMultiplier: 50, // 50% job markup
	}

	res := Calculate(in)

	// Math Breakdown:
	// Raw Filament Cost: 50g * 500 Paisa = 25,000 Paisa (Rs 250.00)
	assert.Equal(t, int64(25000), res.RawFilamentCost)

	// Filament Profit Markup: 25000 Paisa * 100% = 25,000 Paisa (Rs 250.00)
	assert.Equal(t, int64(25000), res.FilamentProfitMarkup)

	// Machine Time Cost: 4.5h * 200 Paisa/h = 900 Paisa (Rs 9.00)
	assert.Equal(t, int64(900), res.TimeCost)

	// Base Production Cost:
	// Filament (25000) + Time (900) + Labor (150) + Electricity (50) + Postproc (100) = 26,200 Paisa (Rs 262.00)
	assert.Equal(t, int64(26200), res.BaseProductionCost)

	// Failure Allowance:
	// 26,200 * 5% = 1310 Paisa (Rs 13.10)
	assert.Equal(t, int64(1310), res.FailureAllowance)

	// Total Production Cost:
	// Base Production (26200) + Failure Allowance (1310) + Packaging (250) + Shipping (500) = 28,260 Paisa (Rs 282.60)
	assert.Equal(t, int64(28260), res.TotalProductionCost)

	// Final Profit Markup:
	// Total Production (28260) * 50% = 14,130 Paisa (Rs 141.30)
	assert.Equal(t, int64(14130), res.FinalProfitMarkup)

	// Final Price:
	// Total Production (28260) + Filament Markup (25000) + Final Profit Markup (14130) = 67,390 Paisa (Rs 673.90)
	assert.Equal(t, int64(67390), res.FinalPrice)
}
