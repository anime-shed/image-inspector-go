package validation

import (
	"net/url"
	"strings"

	apperrors "github.com/anime-shed/image-inspector-go/internal/errors"
)

// URLValidator handles URL validation logic
type URLValidator struct {
	allowedSchemes []string
	allowedHosts   []string
}

// NewURLValidator creates a new URL validator with default settings
func NewURLValidator() *URLValidator {
	return &URLValidator{
		allowedSchemes: []string{"http", "https"},
		allowedHosts:   []string{}, // empty means all hosts allowed
	}
}

// NewURLValidatorWithOptions creates a URL validator with custom options
func NewURLValidatorWithOptions(schemes []string, hosts []string) *URLValidator {
	return &URLValidator{
		allowedSchemes: schemes,
		allowedHosts:   hosts,
	}
}

// ValidateImageURL validates if the provided URL is acceptable for image processing
func (v *URLValidator) ValidateImageURL(imageURL string) error {
	if strings.TrimSpace(imageURL) == "" {
		return apperrors.NewValidationError("URL cannot be empty", nil)
	}

	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return apperrors.NewValidationError("Invalid URL format", err)
	}

	if !v.isSchemeAllowed(parsedURL.Scheme) {
		return apperrors.NewValidationError("URL scheme not allowed", nil)
	}

	if parsedURL.Host == "" {
		return apperrors.NewValidationError("URL must have a valid host", nil)
	}

	if len(v.allowedHosts) > 0 && !v.isHostAllowed(parsedURL.Host) {
		return apperrors.NewValidationError("URL host not allowed", nil)
	}

	return nil
}

// isSchemeAllowed checks if the URL scheme is in the allowed list
func (v *URLValidator) isSchemeAllowed(scheme string) bool {
	for _, allowed := range v.allowedSchemes {
		if scheme == allowed {
			return true
		}
	}
	return false
}

// isHostAllowed checks if the URL host is in the allowed list
// Returns true if no host restrictions are set (empty allowedHosts)
func (v *URLValidator) isHostAllowed(host string) bool {
	if len(v.allowedHosts) == 0 {
		return true
	}
	for _, allowed := range v.allowedHosts {
		if host == allowed {
			return true
		}
	}
	return false
}
