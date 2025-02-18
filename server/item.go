package server

import "regexp"

// isSKU checks if the input string is an SKU
func isSKU(input string) bool {
	// Define a regex pattern for SKUs (alphanumeric, no spaces, 6 characters)
	skuPattern := `^[a-zA-Z0-9]{6,6}$`
	matched, err := regexp.MatchString(skuPattern, input)
	if err != nil {
		return false
	}
	return matched
}
