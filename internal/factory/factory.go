package factory

import (
	"fmt"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/storage"
)

// AnalyzerType represents different types of image analyzers
type AnalyzerType string

const (
	// StandardAnalyzer for general image analysis
	StandardAnalyzer AnalyzerType = "standard"
	// OCRAnalyzer for OCR-specific analysis
	OCRAnalyzer AnalyzerType = "ocr"
	// QualityAnalyzer for quality-focused analysis
	QualityAnalyzer AnalyzerType = "quality"
)

// StorageType represents different types of storage backends
type StorageType string

const (
	// HTTPStorage for HTTP-based image fetching
	HTTPStorage StorageType = "http"
	// AzureStorage for Azure blob storage
	AzureStorage StorageType = "azure"
	// LocalStorage for local file system
	LocalStorage StorageType = "local"
)

// AnalyzerFactory creates image analyzers
type AnalyzerFactory interface {
	CreateAnalyzer(analyzerType AnalyzerType) (analyzer.ImageAnalyzer, error)
}

// StorageFactory creates storage implementations
type StorageFactory interface {
	CreateStorage(storageType StorageType) (storage.ImageFetcher, error)
}

// analyzerFactory implements AnalyzerFactory
type analyzerFactory struct{}

// NewAnalyzerFactory creates a new analyzer factory
func NewAnalyzerFactory() AnalyzerFactory {
	return &analyzerFactory{}
}

// CreateAnalyzer creates an analyzer based on the specified type
func (f *analyzerFactory) CreateAnalyzer(analyzerType AnalyzerType) (analyzer.ImageAnalyzer, error) {
	switch analyzerType {
	case StandardAnalyzer, OCRAnalyzer, QualityAnalyzer:
		// For now, all types use the same implementation
		// In the future, we could have specialized implementations
		return analyzer.NewImageAnalyzer()
	default:
		return nil, fmt.Errorf("unsupported analyzer type: %s", analyzerType)
	}
}

// storageFactory implements StorageFactory
type storageFactory struct{}

// NewStorageFactory creates a new storage factory
func NewStorageFactory() StorageFactory {
	return &storageFactory{}
}

// CreateStorage creates a storage implementation based on the specified type
func (f *storageFactory) CreateStorage(storageType StorageType) (storage.ImageFetcher, error) {
	switch storageType {
	case HTTPStorage:
		return storage.NewHTTPImageFetcher(), nil
	case AzureStorage:
		// TODO: Implement Azure storage when needed
		return nil, fmt.Errorf("azure storage not yet implemented")
	case LocalStorage:
		// TODO: Implement local storage when needed
		return nil, fmt.Errorf("local storage not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// ComponentFactory combines all factories
type ComponentFactory struct {
	AnalyzerFactory AnalyzerFactory
	StorageFactory  StorageFactory
}

// NewComponentFactory creates a new component factory
func NewComponentFactory() *ComponentFactory {
	return &ComponentFactory{
		AnalyzerFactory: NewAnalyzerFactory(),
		StorageFactory:  NewStorageFactory(),
	}
}