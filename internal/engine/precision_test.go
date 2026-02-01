package engine

import (
	"testing"
)

func TestPriceConversion(t *testing.T) {
	tests := []struct {
		name      string
		priceUSDT float64
		expected  int64
	}{
		{"Whole number", 45000.0, 45_000_000_000},
		{"Two decimals", 45000.50, 45_000_500_000},
		{"Six decimals", 45000.123456, 45_000_123_456},
		{"Small price", 0.000001, 1},
		{"Large price", 100000.0, 100_000_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PriceToInt(tt.priceUSDT)
			if result != tt.expected {
				t.Errorf("PriceToInt(%f) = %d, want %d", tt.priceUSDT, result, tt.expected)
			}

			// Test reverse conversion
			reverse := PriceFromInt(result)
			if reverse != tt.priceUSDT {
				t.Errorf("PriceFromInt(%d) = %f, want %f", result, reverse, tt.priceUSDT)
			}
		})
	}
}

func TestQuantityConversion(t *testing.T) {
	tests := []struct {
		name     string
		quantity float64
		expected int64
	}{
		{"Half unit", 0.5, 50_000_000},
		{"One unit", 1.0, 100_000_000},
		{"Eight decimals", 0.12345678, 12_345_678},
		{"Small amount", 0.00000001, 1}, // Smallest possible unit
		{"Large amount", 100.0, 10_000_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuantityToInt(tt.quantity)
			if result != tt.expected {
				t.Errorf("QuantityToInt(%f) = %d, want %d", tt.quantity, result, tt.expected)
			}

			// Test reverse conversion
			reverse := QuantityFromInt(result)
			if reverse != tt.quantity {
				t.Errorf("QuantityFromInt(%d) = %f, want %f", result, reverse, tt.quantity)
			}
		})
	}
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		price    int64
		expected string
	}{
		{45_000_000_000, "$45000.00"},
		{45_000_500_000, "$45000.50"},
		{123_456_789_000, "$123456.79"},
	}

	for _, tt := range tests {
		result := FormatPrice(tt.price)
		if result != tt.expected {
			t.Errorf("FormatPrice(%d) = %s, want %s", tt.price, result, tt.expected)
		}
	}
}

func TestCalculateNotional(t *testing.T) {
	// Test: Buy 0.5 BTC at $45,000 = $22,500
	price := PriceToInt(45000.0)   // 45,000 USDT
	quantity := QuantityToInt(0.5) // 0.5 BTC
	notional := CalculateNotional(price, quantity)

	expected := 22500.0
	if notional != expected {
		t.Errorf("CalculateNotional() = %f, want %f", notional, expected)
	}
}

func TestCalculateNotionalInt(t *testing.T) {
	// Test: Buy 0.5 BTC at $45,000 = $22,500 (in micro-USDT)
	price := PriceToInt(45000.0)   // 45,000 USDT
	quantity := QuantityToInt(0.5) // 0.5 BTC
	notional := CalculateNotionalInt(price, quantity)

	expected := PriceToInt(22500.0)
	if notional != expected {
		t.Errorf("CalculateNotionalInt() = %d, want %d", notional, expected)
	}
}

func TestValidation(t *testing.T) {
	// Valid price
	if err := ValidatePrice(PriceToInt(100.0)); err != nil {
		t.Errorf("ValidatePrice failed for valid price: %v", err)
	}

	// Invalid price (negative)
	if err := ValidatePrice(-100); err == nil {
		t.Error("ValidatePrice should fail for negative price")
	}

	// Invalid price (zero)
	if err := ValidatePrice(0); err == nil {
		t.Error("ValidatePrice should fail for zero price")
	}

	// Valid quantity
	if err := ValidateQuantity(QuantityToInt(1.0)); err != nil {
		t.Errorf("ValidateQuantity failed for valid quantity: %v", err)
	}

	// Invalid quantity (negative)
	if err := ValidateQuantity(-100); err == nil {
		t.Error("ValidateQuantity should fail for negative quantity")
	}
}

// Example test showing real-world usage
func TestRealWorldExample(t *testing.T) {
	// Scenario: User wants to buy 0.25 BTC at $45,123.50

	// Convert user input to integers
	price := PriceToInt(45123.50)
	quantity := QuantityToInt(0.25)

	// Validate inputs
	if err := ValidatePrice(price); err != nil {
		t.Fatalf("Invalid price: %v", err)
	}
	if err := ValidateQuantity(quantity); err != nil {
		t.Fatalf("Invalid quantity: %v", err)
	}

	// Create an order
	order := NewOrder(true, quantity)
	if order.Size != quantity {
		t.Errorf("Order size = %d, want %d", order.Size, quantity)
	}

	// Calculate total cost
	totalUSDT := CalculateNotional(price, quantity)
	expectedTotal := 45123.50 * 0.25 // = 11280.875
	if totalUSDT != expectedTotal {
		t.Errorf("Total USDT = %f, want %f", totalUSDT, expectedTotal)
	}

	// Display to user
	t.Logf("Order Details:")
	t.Logf("  Price: %s", FormatPrice(price))
	t.Logf("  Quantity: %s BTC", FormatQuantity(quantity))
	t.Logf("  Total: $%.2f", totalUSDT)
}
