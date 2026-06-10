package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateLuhnAlgorithm(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{"valid Luhn", "9278923470", true},
		{"valid Luhn Visa", "4111111111111111", true},
		{"valid Luhn MasterCard", "5555555555554444", true},
		{"invalid Luhn", "4532017906421308", false},
		{"too short", "12", false},
		{"contains letter", "4532a0179", false},
		{"empty string", "", false},
		{"with spaces removed", " 4111 1111 1111 1111 ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateLuhnAlgorithm(tt.number)
			assert.Equal(t, tt.want, got)
		})
	}
}
