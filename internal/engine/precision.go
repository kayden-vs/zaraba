package engine

import (
	"fmt"
	"math"
)

// Precision constants for the matching engine
const (
	// PriceScale: 1 USDT = 1,000,000 micro-USDT
	// This gives us 6 decimal places for price precision
	// Example: $45,000.123456 USDT = 45,000,123,456 units
	PriceScale int64 = 1_000_000

	// QuantityScale: 1 unit = 100,000,000 base units
	// This gives us 8 decimal places for quantity (like Bitcoin satoshis)
	// Example: 0.12345678 BTC = 12,345,678 units
	QuantityScale int64 = 100_000_000
)

// Helper functions for converting between human-readable values and internal integers

// PriceToInt converts a price in USDT (as float64) to micro-USDT (int64)
// Example: PriceToInt(45000.50) = 45_000_500_000
func PriceToInt(price float64) int64 {
	return int64(math.Round(price * float64(PriceScale)))
}

// PriceFromInt converts a price in micro-USDT (int64) to USDT (float64)
// Example: PriceFromInt(45_000_500_000) = 45000.50
func PriceFromInt(price int64) float64 {
	return float64(price) / float64(PriceScale)
}

// QuantityToInt converts a quantity (as float64) to base units (int64)
// Example: QuantityToInt(0.5) = 50_000_000
func QuantityToInt(quantity float64) int64 {
	return int64(math.Round(quantity * float64(QuantityScale)))
}

// QuantityFromInt converts a quantity in base units (int64) to decimal (float64)
// Example: QuantityFromInt(50_000_000) = 0.5
func QuantityFromInt(quantity int64) float64 {
	return float64(quantity) / float64(QuantityScale)
}

// FormatPrice formats a price (in micro-USDT) as a human-readable string
// Example: FormatPrice(45_000_500_000) = "$45,000.50"
func FormatPrice(price int64) string {
	return fmt.Sprintf("$%.2f", PriceFromInt(price))
}

// FormatQuantity formats a quantity (in base units) as a human-readable string
// Example: FormatQuantity(50_000_000) = "0.50000000"
func FormatQuantity(quantity int64) string {
	return fmt.Sprintf("%.8f", QuantityFromInt(quantity))
}

// CalculateNotional calculates the total value of a trade in USDT
// Takes price (micro-USDT) and quantity (base units), returns USDT value
// Example: CalculateNotional(45_000_000_000, 50_000_000) = 22500.0
func CalculateNotional(price, quantity int64) float64 {
	// Both are integers, we need to convert properly
	// notional = (price / PriceScale) * (quantity / QuantityScale)
	// To avoid precision loss, we do: (price * quantity) / (PriceScale * QuantityScale)
	priceFloat := PriceFromInt(price)
	quantityFloat := QuantityFromInt(quantity)
	return priceFloat * quantityFloat
}

// CalculateNotionalInt calculates the total value in micro-USDT (all integers)
// Returns the notional value in micro-USDT
func CalculateNotionalInt(price, quantity int64) int64 {
	// notional_micro_usdt = (price_micro_usdt * quantity_base_units) / QuantityScale
	// We need to be careful about overflow for very large trades
	return (price * quantity) / QuantityScale
}

// ValidatePrice checks if a price is valid (positive and within reasonable bounds)
func ValidatePrice(price int64) error {
	if price <= 0 {
		return fmt.Errorf("price must be positive, got %d", price)
	}
	// Max price: $1,000,000 per unit (to prevent overflow)
	maxPrice := PriceToInt(1_000_000)
	if price > maxPrice {
		return fmt.Errorf("price too large: %s (max: $1,000,000)", FormatPrice(price))
	}
	return nil
}

// ValidateQuantity checks if a quantity is valid (positive)
func ValidateQuantity(quantity int64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive, got %d", quantity)
	}
	return nil
}
