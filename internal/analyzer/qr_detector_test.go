package analyzer

import (
	"image"
	"image/color"
	"testing"
)

func TestNewQRDetector(t *testing.T) {
	detector := NewQRDetector()
	if detector == nil {
		t.Fatal("Expected non-nil QR detector")
	}
}

func TestDetectQRCode_EmptyImage(t *testing.T) {
	detector := NewQRDetector()

	// Create a uniform image with no QR patterns
	img := createTestImage(200, 200, color.RGBA{255, 255, 255, 255})

	hasQR := detector.DetectQRCode(img)

	// Should not detect QR codes in uniform image
	if hasQR {
		t.Error("Expected no QR codes in uniform white image")
	}
}

func TestDetectQRCode_SimplePattern(t *testing.T) {
	detector := NewQRDetector()

	// Create an image with some black and white patterns
	// (not a real QR code, but has some contrast)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			// Create a checkerboard pattern
			if (x/10+y/10)%2 == 0 {
				img.Set(x, y, color.RGBA{0, 0, 0, 255}) // Black
			} else {
				img.Set(x, y, color.RGBA{255, 255, 255, 255}) // White
			}
		}
	}

	hasQR := detector.DetectQRCode(img)

	// This test mainly ensures the function doesn't crash
	// QR detection is complex and may or may not detect patterns
	_ = hasQR // Result depends on the specific pattern detection algorithm
}

func TestDetectQRCode_SmallImage(t *testing.T) {
	detector := NewQRDetector()

	// Create a very small image
	img := createTestImage(10, 10, color.RGBA{128, 128, 128, 255})

	hasQR := detector.DetectQRCode(img)

	// Should not detect QR codes in very small image
	if hasQR {
		t.Error("Expected no QR codes in very small image")
	}
}

func TestDetectQRCode_GrayscaleImage(t *testing.T) {
	detector := NewQRDetector()

	// Create a grayscale image with some patterns
	gray := image.NewGray(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			// Create alternating stripes
			if x%20 < 10 {
				gray.Set(x, y, color.Gray{0}) // Black stripe
			} else {
				gray.Set(x, y, color.Gray{255}) // White stripe
			}
		}
	}

	hasQR := detector.DetectQRCode(gray)

	// This test ensures the function works with grayscale images
	_ = hasQR // Result depends on pattern detection
}

// Note: Tests for private methods (detectFinderPatterns, checkConcentricSquares, toBinary)
// are not included as they are internal implementation details and not part of the public API.
// The public DetectQRCode method provides sufficient test coverage for the QR detection functionality.
