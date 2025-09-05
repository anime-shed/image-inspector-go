package repository

import "errors"

var (
	// ErrInvalidImageURL indicates an invalid image URL
	ErrInvalidImageURL = errors.New("invalid image URL")

	// ErrImageNotFound indicates the image was not found
	ErrImageNotFound = errors.New("image not found")

	// ErrAnalysisNotFound indicates the analysis result was not found
	ErrAnalysisNotFound = errors.New("analysis result not found")

	// ErrRepositoryUnavailable indicates the repository is unavailable
	ErrRepositoryUnavailable = errors.New("repository unavailable")
)
