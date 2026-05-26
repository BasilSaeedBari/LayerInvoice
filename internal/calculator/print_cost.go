package calculator

import (
	"math"
)

// PrintJobInputs contains all variables required to calculate the cost of a 3D print.
type PrintJobInputs struct {
	Grams             float64 // Grams of filament used
	CostPerGram       int64   // Cost of filament per gram (smallest currency unit, e.g. Paisa)
	FilamentProfitPct int64   // Profit percentage on filament (e.g. 100 = 100% markup)

	PrintHours      float64 // Print time in hours
	TimeRate        int64   // Machine/hourly rate (smallest currency unit per hour)
	LabourCost      int64   // Personal labor cost (smallest currency unit)
	ElectricityCost int64   // Electricity cost (smallest currency unit)
	PostprocCost    int64   // Post-processing / cleanup cost (smallest currency unit)

	PackagingCost int64 // Packaging cost (smallest currency unit)
	ShippingCost  int64 // Shipping cost (smallest currency unit)

	FailureRate      int64 // Expected failure rate percentage (e.g. 5 = 5%)
	ProfitMultiplier int64 // Profit markup percentage on total production cost (e.g. 50 = 50% markup)
}

// PrintJobBreakdown contains the detailed results of the print job cost calculations.
type PrintJobBreakdown struct {
	RawFilamentCost      int64 // raw weight * cost per gram
	FilamentProfitMarkup int64 // markup on filament only
	TimeCost             int64 // hours * hourly rate
	BaseProductionCost   int64 // sum of filament cost, time, labor, electricity, postproc
	FailureAllowance     int64 // base production cost * failure rate
	TotalProductionCost  int64 // base production cost + failure allowance + packaging + shipping
	FinalProfitMarkup    int64 // total production cost * final profit multiplier
	FinalPrice           int64 // total production cost + filament markup + final profit markup
}

// RoundHalfUp rounds a float64 value to the nearest int64 using standard ROUND_HALF_UP logic.
func RoundHalfUp(val float64) int64 {
	if val < 0 {
		return int64(math.Ceil(val - 0.5))
	}
	return int64(math.Floor(val + 0.5))
}

// Calculate computes the comprehensive print job cost breakdown.
func Calculate(in PrintJobInputs) PrintJobBreakdown {
	// 1. Raw Filament Cost
	rawFilamentCost := RoundHalfUp(in.Grams * float64(in.CostPerGram))

	// 2. Filament Profit Markup
	filamentProfitMarkup := RoundHalfUp(float64(rawFilamentCost) * float64(in.FilamentProfitPct) / 100.0)

	// 3. Machine Time Cost
	timeCost := RoundHalfUp(in.PrintHours * float64(in.TimeRate))

	// 4. Base Production Cost
	baseProductionCost := rawFilamentCost + timeCost + in.LabourCost + in.ElectricityCost + in.PostprocCost

	// 5. Failure Allowance (applies to base manufacturing costs: filament, time, labor, electricity, postprocessing)
	failureAllowance := RoundHalfUp(float64(baseProductionCost) * float64(in.FailureRate) / 100.0)

	// 6. Total Production Cost (adds packaging and shipping which are not wasted in failure)
	totalProductionCost := baseProductionCost + failureAllowance + in.PackagingCost + in.ShippingCost

	// 7. Final Profit Markup (applied on total production cost)
	finalProfitMarkup := RoundHalfUp(float64(totalProductionCost) * float64(in.ProfitMultiplier) / 100.0)

	// 8. Final Price
	finalPrice := totalProductionCost + filamentProfitMarkup + finalProfitMarkup

	return PrintJobBreakdown{
		RawFilamentCost:      rawFilamentCost,
		FilamentProfitMarkup: filamentProfitMarkup,
		TimeCost:             timeCost,
		BaseProductionCost:   baseProductionCost,
		FailureAllowance:     failureAllowance,
		TotalProductionCost:  totalProductionCost,
		FinalProfitMarkup:    finalProfitMarkup,
		FinalPrice:           finalPrice,
	}
}
