package repository

import (
	"context"
	"image"
)

// ImageRepository defines the interface for image data access operations
type ImageRepository interface {
	// FetchImage retrieves an image from a URL
	FetchImage(ctx context.Context, imageURL string) (image.Image, error)
	
	// ValidateImageURL validates if the provided URL is acceptable
	ValidateImageURL(imageURL string) error
	
	// GetImageMetadata retrieves metadata about an image without downloading it
	GetImageMetadata(ctx context.Context, imageURL string) (*ImageMetadata, error)
}

// ImageMetadata contains metadata about an image
type ImageMetadata struct {
	ContentType   string
	ContentLength int64
	Width         int
	Height        int
	Format        string
}

// AnalysisRepository defines the interface for analysis result operations
type AnalysisRepository interface {
	// SaveAnalysisResult stores an analysis result
	SaveAnalysisResult(ctx context.Context, result *AnalysisResult) error
	
	// GetAnalysisResult retrieves a stored analysis result
	GetAnalysisResult(ctx context.Context, id string) (*AnalysisResult, error)
	
	// GetAnalysisHistory retrieves analysis history for a specific image URL
	GetAnalysisHistory(ctx context.Context, imageURL string) ([]*AnalysisResult, error)
}

// AnalysisResult represents the result of an image analysis
type AnalysisResult struct {
	ID                string    `json:"id"`
	ImageURL          string    `json:"image_url"`
	Timestamp         string    `json:"timestamp"`
	ProcessingTimeSec float64   `json:"processing_time_sec"`
	Overexposed       bool      `json:"overexposed"`
	Oversaturated     bool      `json:"oversaturated"`
	IncorrectWB       bool      `json:"incorrect_wb"`
	Blurry            bool      `json:"blurry"`
	LaplacianVar      float64   `json:"laplacian_var"`
	AvgLuminance      float64   `json:"avg_luminance"`
	AvgSaturation     float64   `json:"avg_saturation"`
	ChannelBalance    [3]float64 `json:"channel_balance"`
	Errors            []string  `json:"errors,omitempty"`
}