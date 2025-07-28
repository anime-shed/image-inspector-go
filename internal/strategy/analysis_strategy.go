package strategy

import (
	"image"
	"go-image-inspector/internal/analyzer"
)

// AnalysisStrategy defines the interface for different analysis strategies
type AnalysisStrategy interface {
	Analyze(img image.Image) analyzer.AnalysisResult
	GetStrategyName() string
}

// QualityAnalysisStrategy focuses on image quality assessment
type QualityAnalysisStrategy struct {
	analyzer analyzer.ImageAnalyzer
}

// NewQualityAnalysisStrategy creates a new quality analysis strategy
func NewQualityAnalysisStrategy(analyzer analyzer.ImageAnalyzer) AnalysisStrategy {
	return &QualityAnalysisStrategy{
		analyzer: analyzer,
	}
}

// Analyze performs quality-focused analysis
func (s *QualityAnalysisStrategy) Analyze(img image.Image) analyzer.AnalysisResult {
	return s.analyzer.Analyze(img, false)
}

// GetStrategyName returns the strategy name
func (s *QualityAnalysisStrategy) GetStrategyName() string {
	return "quality_analysis"
}

// OCRAnalysisStrategy focuses on OCR-specific analysis
type OCRAnalysisStrategy struct {
	analyzer analyzer.ImageAnalyzer
}

// NewOCRAnalysisStrategy creates a new OCR analysis strategy
func NewOCRAnalysisStrategy(analyzer analyzer.ImageAnalyzer) AnalysisStrategy {
	return &OCRAnalysisStrategy{
		analyzer: analyzer,
	}
}

// Analyze performs OCR-focused analysis
func (s *OCRAnalysisStrategy) Analyze(img image.Image) analyzer.AnalysisResult {
	return s.analyzer.Analyze(img, true)
}

// GetStrategyName returns the strategy name
func (s *OCRAnalysisStrategy) GetStrategyName() string {
	return "ocr_analysis"
}

// FastAnalysisStrategy provides quick analysis with reduced accuracy
type FastAnalysisStrategy struct {
	analyzer analyzer.ImageAnalyzer
}

// NewFastAnalysisStrategy creates a new fast analysis strategy
func NewFastAnalysisStrategy(analyzer analyzer.ImageAnalyzer) AnalysisStrategy {
	return &FastAnalysisStrategy{
		analyzer: analyzer,
	}
}

// Analyze performs fast analysis
func (s *FastAnalysisStrategy) Analyze(img image.Image) analyzer.AnalysisResult {
	// For fast analysis, we use standard mode but could optimize further
	return s.analyzer.Analyze(img, false)
}

// GetStrategyName returns the strategy name
func (s *FastAnalysisStrategy) GetStrategyName() string {
	return "fast_analysis"
}

// AnalysisContext manages the analysis strategy
type AnalysisContext struct {
	strategy AnalysisStrategy
}

// NewAnalysisContext creates a new analysis context
func NewAnalysisContext(strategy AnalysisStrategy) *AnalysisContext {
	return &AnalysisContext{
		strategy: strategy,
	}
}

// SetStrategy changes the analysis strategy
func (c *AnalysisContext) SetStrategy(strategy AnalysisStrategy) {
	c.strategy = strategy
}

// ExecuteAnalysis performs analysis using the current strategy
func (c *AnalysisContext) ExecuteAnalysis(img image.Image) analyzer.AnalysisResult {
	return c.strategy.Analyze(img)
}

// GetCurrentStrategy returns the current strategy name
func (c *AnalysisContext) GetCurrentStrategy() string {
	return c.strategy.GetStrategyName()
}