package utils

import (
	"strings"
)

// checks if a given string satisfies the Luhn algorithm.
func ValidateLuhnAlgorithm(number string) bool {
	// Remove any spaces from the input string
	number = strings.ReplaceAll(number, " ", "")
	// Luhn numbers must be at least 2 digits long
	if len(number) < 2 {
		return false
	}
	sum := 0
	shouldDouble := false
	// Loop from the rightmost digit to the left
	for i := len(number) - 1; i >= 0; i-- {
		r := number[i]
		// Ensure the character is a valid digit
		if r < '0' || r > '9' {
			return false
		}
		digit := int(r - '0')
		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		shouldDouble = !shouldDouble
	}
	// The identifier is valid if the sum is a multiple of 10
	return sum%10 == 0
}
