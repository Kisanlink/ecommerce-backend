package utils

import (
	"math"
)

// CurrencyUtils provides utility functions for handling currency calculations
// to avoid floating-point precision issues

// RoundToTwoDecimals rounds a float64 to 2 decimal places
func RoundToTwoDecimals(amount float64) float64 {
	return math.Round(amount*100) / 100
}

// RupeesToPaisa converts rupees to paisa (smallest currency unit)
// Uses proper rounding to avoid floating-point precision issues
func RupeesToPaisa(rupees float64) int {
	// Round to 2 decimal places first to avoid floating-point precision issues
	roundedRupees := RoundToTwoDecimals(rupees)
	return int(math.Round(roundedRupees * 100))
}

// PaisaToRupees converts paisa to rupees
func PaisaToRupees(paisa int) float64 {
	return RoundToTwoDecimals(float64(paisa) / 100.0)
}

// CalculateCartItemTotal calculates the total price for a cart item
// with proper rounding to avoid floating-point precision issues
func CalculateCartItemTotal(quantity int, unitPrice float64) float64 {
	// Round unit price to 2 decimal places first
	roundedUnitPrice := RoundToTwoDecimals(unitPrice)
	return RoundToTwoDecimals(float64(quantity) * roundedUnitPrice)
}

// ApplyDiscount applies a discount to an amount with proper rounding
func ApplyDiscount(amount, discount float64) float64 {
	roundedAmount := RoundToTwoDecimals(amount)
	roundedDiscount := RoundToTwoDecimals(discount)

	result := roundedAmount - roundedDiscount
	if result < 0 {
		return 0.0
	}
	return RoundToTwoDecimals(result)
}

// AddShippingCharges adds shipping charges to an amount with proper rounding
func AddShippingCharges(amount, shipping float64) float64 {
	roundedAmount := RoundToTwoDecimals(amount)
	roundedShipping := RoundToTwoDecimals(shipping)
	return RoundToTwoDecimals(roundedAmount + roundedShipping)
}
