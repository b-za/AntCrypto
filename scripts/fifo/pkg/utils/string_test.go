package utils

import "testing"

func TestHasPrefixCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected bool
	}{
		{"Match lowercase", "bought btc", "bought", true},
		{"Match uppercase", "BOUGHT BTC", "bought", true},
		{"Match mixed case", "BoUgHt BTC", "bought", true},
		{"No match", "sold btc", "bought", false},
		{"Empty prefix", "btc", "bought", false},
		{"Fee matching", "Trading fee", "trading", true},
		{"Fee matching capital F", "Trading Fee", "trading", true},
		{"Fee matching all caps", "TRADING FEE", "trading", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPrefixCaseInsensitive(tt.s, tt.prefix)
			if result != tt.expected {
				t.Errorf("HasPrefixCaseInsensitive(%q, %q) = %v, want %v",
					tt.s, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestEqualCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected bool
	}{
		{"Match lowercase", "zar", "zar", true},
		{"Match uppercase", "ZAR", "zar", true},
		{"Match mixed case", "ZAr", "zar", true},
		{"No match", "xbt", "zar", false},
		{"Different strings", "zarr", "zar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EqualCaseInsensitive(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("EqualCaseInsensitive(%q, %q) = %v, want %v",
					tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestContainsCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"Contains substring", "Bought BTC for ZAR", "btc", true},
		{"Case insensitive", "bought btc for zar", "BTC", true},
		{"Not contains", "Bought ETH for ZAR", "btc", false},
		{"Empty substring", "Bought BTC", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsCaseInsensitive(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("ContainsCaseInsensitive(%q, %q) = %v, want %v",
					tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestTrimSpaceAndCase(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedOriginal   string
		expectedNormalized string
	}{
		{"Trim spaces", "  Bought BTC  ", "Bought BTC", "bought btc"},
		{"Mixed case", "  BoUgHt BTC  ", "BoUgHt BTC", "bought btc"},
		{"Already clean", "Bought BTC", "Bought BTC", "bought btc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, normalized := TrimSpaceAndCase(tt.input)
			if original != tt.expectedOriginal {
				t.Errorf("TrimSpaceAndCase(%q) original = %q, want %q",
					tt.input, original, tt.expectedOriginal)
			}
			if normalized != tt.expectedNormalized {
				t.Errorf("TrimSpaceAndCase(%q) normalized = %q, want %q",
					tt.input, normalized, tt.expectedNormalized)
			}
		})
	}
}
