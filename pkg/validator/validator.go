package validator

import (
	"net/url"
	"strings"
)

// ValidateURL checks if a URL is valid
func ValidateURL(urlStr string) error {
	// Trim whitespace
	urlStr = strings.TrimSpace(urlStr)

	if urlStr == "" {
		return ErrEmptyURL
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ErrInvalidURL
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrInvalidScheme
	}

	// Check host
	if parsedURL.Host == "" {
		return ErrInvalidHost
	}

	return nil
}

// ValidateCustomAlias checks if a custom alias is valid
func ValidateCustomAlias(alias string) error {
	if len(alias) < 3 || len(alias) > 20 {
		return ErrInvalidAliasLength
	}

	// Check if alphanumeric with hyphens and underscores
	for _, char := range alias {
		if !isAlphanumeric(char) && char != '-' && char != '_' {
			return ErrInvalidAliasFormat
		}
	}

	return nil
}

func isAlphanumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9')
}
