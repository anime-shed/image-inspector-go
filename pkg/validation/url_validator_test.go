package validation

import (
	"testing"
	apperrors "go-image-inspector/internal/errors"
)

func TestNewURLValidator(t *testing.T) {
	validator := NewURLValidator()
	if validator == nil {
		t.Fatal("Expected non-nil URL validator")
	}

	// Check default schemes
	expectedSchemes := []string{"http", "https"}
	if len(validator.allowedSchemes) != len(expectedSchemes) {
		t.Errorf("Expected %d schemes, got %d", len(expectedSchemes), len(validator.allowedSchemes))
	}

	for i, scheme := range expectedSchemes {
		if validator.allowedSchemes[i] != scheme {
			t.Errorf("Expected scheme %s, got %s", scheme, validator.allowedSchemes[i])
		}
	}
}

func TestNewURLValidatorWithOptions(t *testing.T) {
	schemes := []string{"https"}
	hosts := []string{"example.com", "test.com"}
	validator := NewURLValidatorWithOptions(schemes, hosts)

	if len(validator.allowedSchemes) != 1 || validator.allowedSchemes[0] != "https" {
		t.Error("Expected only https scheme")
	}

	if len(validator.allowedHosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(validator.allowedHosts))
	}
}

func TestValidateImageURL_ValidURLs(t *testing.T) {
	validator := NewURLValidator()

	validURLs := []string{
		"http://example.com/image.jpg",
		"https://example.com/image.png",
		"https://subdomain.example.com/path/to/image.gif",
		"http://192.168.1.1/image.jpg",
	}

	for _, url := range validURLs {
		err := validator.ValidateImageURL(url)
		if err != nil {
			t.Errorf("Expected valid URL %s to pass validation, got error: %v", url, err)
		}
	}
}

func TestValidateImageURL_EmptyURL(t *testing.T) {
	validator := NewURLValidator()

	emptyURLs := []string{
		"",
		"   ",
		"\t\n",
	}

	for _, url := range emptyURLs {
		err := validator.ValidateImageURL(url)
		if err == nil {
			t.Errorf("Expected empty URL '%s' to fail validation", url)
		}

		if appErr, ok := err.(*apperrors.AppError); ok {
			if appErr.Message != "URL cannot be empty" {
				t.Errorf("Expected 'URL cannot be empty' error, got: %s", appErr.Message)
			}
		} else {
			t.Errorf("Expected AppError, got: %T", err)
		}
	}
}

func TestValidateImageURL_InvalidFormat(t *testing.T) {
	validator := NewURLValidator()

	invalidURLs := []string{
		"not-a-url",
		"://missing-scheme",
		"http://",
		"ftp://example.com", // invalid scheme
	}

	for _, url := range invalidURLs {
		err := validator.ValidateImageURL(url)
		if err == nil {
			t.Errorf("Expected invalid URL '%s' to fail validation", url)
		}
	}
}

func TestValidateImageURL_NoHost(t *testing.T) {
	validator := NewURLValidator()

	noHostURLs := []string{
		"http://",
		"https://",
		"http:///path",
	}

	for _, url := range noHostURLs {
		err := validator.ValidateImageURL(url)
		if err == nil {
			t.Errorf("Expected URL without host '%s' to fail validation", url)
		}

		if appErr, ok := err.(*apperrors.AppError); ok {
			if appErr.Message != "URL must have a valid host" {
				t.Errorf("Expected 'URL must have a valid host' error, got: %s", appErr.Message)
			}
		}
	}
}

func TestValidateImageURL_InvalidScheme(t *testing.T) {
	validator := NewURLValidator()

	invalidSchemeURLs := []string{
		"ftp://example.com/image.jpg",
		"file://local/path/image.jpg",
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
	}

	for _, url := range invalidSchemeURLs {
		err := validator.ValidateImageURL(url)
		if err == nil {
			t.Errorf("Expected URL with invalid scheme '%s' to fail validation", url)
		}

		if appErr, ok := err.(*apperrors.AppError); ok {
			if appErr.Message != "URL scheme not allowed" {
				t.Errorf("Expected 'URL scheme not allowed' error, got: %s", appErr.Message)
			}
		}
	}
}

func TestValidateImageURL_RestrictedHosts(t *testing.T) {
	allowedHosts := []string{"example.com", "trusted.com"}
	validator := NewURLValidatorWithOptions([]string{"http", "https"}, allowedHosts)

	// Test allowed hosts
	allowedURLs := []string{
		"http://example.com/image.jpg",
		"https://trusted.com/image.png",
	}

	for _, url := range allowedURLs {
		err := validator.ValidateImageURL(url)
		if err != nil {
			t.Errorf("Expected allowed host URL '%s' to pass validation, got error: %v", url, err)
		}
	}

	// Test disallowed hosts
	disallowedURLs := []string{
		"http://malicious.com/image.jpg",
		"https://untrusted.com/image.png",
	}

	for _, url := range disallowedURLs {
		err := validator.ValidateImageURL(url)
		if err == nil {
			t.Errorf("Expected disallowed host URL '%s' to fail validation", url)
		}

		if appErr, ok := err.(*apperrors.AppError); ok {
			if appErr.Message != "URL host not allowed" {
				t.Errorf("Expected 'URL host not allowed' error, got: %s", appErr.Message)
			}
		}
	}
}

func TestIsSchemeAllowed(t *testing.T) {
	validator := NewURLValidator()

	// Test allowed schemes
	if !validator.isSchemeAllowed("http") {
		t.Error("Expected http scheme to be allowed")
	}
	if !validator.isSchemeAllowed("https") {
		t.Error("Expected https scheme to be allowed")
	}

	// Test disallowed schemes
	if validator.isSchemeAllowed("ftp") {
		t.Error("Expected ftp scheme to be disallowed")
	}
	if validator.isSchemeAllowed("file") {
		t.Error("Expected file scheme to be disallowed")
	}
}

func TestIsHostAllowed(t *testing.T) {
	// Test with no host restrictions (empty allowedHosts)
	validator := NewURLValidator()
	if !validator.isHostAllowed("example.com") {
		t.Error("Expected any host to be allowed when no restrictions")
	}

	// Test with host restrictions
	allowedHosts := []string{"example.com", "trusted.com"}
	restrictedValidator := NewURLValidatorWithOptions([]string{"http", "https"}, allowedHosts)

	if !restrictedValidator.isHostAllowed("example.com") {
		t.Error("Expected example.com to be allowed")
	}
	if !restrictedValidator.isHostAllowed("trusted.com") {
		t.Error("Expected trusted.com to be allowed")
	}
	if restrictedValidator.isHostAllowed("malicious.com") {
		t.Error("Expected malicious.com to be disallowed")
	}
}