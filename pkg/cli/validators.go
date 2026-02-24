package cli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func isNotEmpty(str string) error {
	if str == "" || len(strings.TrimSpace(str)) == 0 {
		return fmt.Errorf("value is required")
	}
	return nil
}

func isValidHost(str string) error {
	if str == "" {
		return fmt.Errorf("host is required")
	}
	// Basic validation for hostname format
	if len(str) > 253 { // Max length of a hostname
		return fmt.Errorf("host name too long")
	}
	// Check if hostname contains only valid characters
	for _, char := range str {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '.' && char != '-' {
			return fmt.Errorf("host contains invalid characters")
		}
	}
	// Check that hostname doesn't start or end with dot/hyphen
	if str[0] == '.' || str[0] == '-' || str[len(str)-1] == '.' || str[len(str)-1] == '-' {
		return fmt.Errorf("host cannot start or end with dots or hyphens")
	}
	return nil
}

func isValidPort(str string) error {
	if str == "" {
		return nil // Allow empty string if you want port to be optional
	}
	port, err := strconv.Atoi(str)
	if err != nil {
		return fmt.Errorf("port must be a number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}
