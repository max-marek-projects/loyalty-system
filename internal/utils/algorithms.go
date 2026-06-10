package utils

import (
	"strings"
)

// checks if a given string satisfies the Luhn algorithm.
func ValidateLuhnAlgorithm(number string) bool {
	number = strings.ReplaceAll(number, " ", "")
	if len(number) < 2 {
		return false
	}
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		ch := number[i]
		if ch < '0' || ch > '9' {
			return false
		}
		digit := int(ch - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}
