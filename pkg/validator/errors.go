package validator

import "errors"

var (
	ErrEmptyURL           = errors.New("URL cannot be empty")
	ErrInvalidURL         = errors.New("invalid URL format")
	ErrInvalidScheme      = errors.New("URL must use http or https scheme")
	ErrInvalidHost        = errors.New("URL must have a valid host")
	ErrInvalidAliasLength = errors.New("custom alias must be 3-20 characters")
	ErrInvalidAliasFormat = errors.New("custom alias must be alphanumeric with optional hyphens and underscores")
)
