package models

// DetailedAnalysisResponse represents a comprehensive image analysis response
// with all individual quality metrics, raw values, thresholds, and detailed breakdowns
type DetailedAnalysisResponse struct {
	// Basic response information
	ImageURL          string    `json:"image_url"`
	Timestamp         string    `json:"timestamp"`
	ProcessingTimeSec float64   `json:"processing_time_sec"`
	
	// Image metadata
	ImageMetadata ImageMetadata `json:"image_metadata"`
	
	// Comprehensive quality analysis
	QualityAnalysis QualityAnalysis `json:"quality_analysis"`
	
	// Raw metrics with detailed calculations
	RawMetrics RawMetrics `json:"raw_metrics"`
	
	// Applied thresholds and parameters
	Thresholds AppliedThresholds `json:"applied_thresholds"`
	
	// Individual quality checks with detailed results
	QualityChecks []QualityCheckResult `json:"quality_checks"`
	
	// OCR specific analysis (if applicable)
	OCRAnalysis *DetailedOCRAnalysis `json:"ocr_analysis,omitempty"`
	
	// Overall assessment
	OverallAssessment OverallAssessment `json:"overall_assessment"`
	
	// Processing details
	ProcessingDetails ProcessingDetails `json:"processing_details"`
	
	// Validation errors and warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// QualityAnalysis provides comprehensive quality assessment
type QualityAnalysis struct {
	// Individual boolean results
	Overexposed      bool `json:"overexposed"`
	Oversaturated    bool `json:"oversaturated"`
	IncorrectWB      bool `json:"incorrect_white_balance"`
	Blurry           bool `json:"blurry"`
	IsLowResolution  bool `json:"is_low_resolution"`
	IsTooDark        bool `json:"is_too_dark"`
	IsTooBright      bool `json:"is_too_bright"`
	IsSkewed         bool `json:"is_skewed"`
	HasDocumentEdges bool `json:"has_document_edges"`
	QRDetected       bool `json:"qr_detected"`
	
	// Overall quality flags
	IsValid          bool `json:"is_valid"`
	IsOCRReady       bool `json:"is_ocr_ready"`
	HasCriticalIssues bool `json:"has_critical_issues"`
	
	// Quality scores (0-100)
	OverallQualityScore float64 `json:"overall_quality_score"`
	SharpnessScore      float64 `json:"sharpness_score"`
	ExposureScore       float64 `json:"exposure_score"`
	ColorScore          float64 `json:"color_score"`
}

// RawMetrics contains all calculated raw values
type RawMetrics struct {
	// Sharpness metrics
	LaplacianVariance    float64 `json:"laplacian_variance"`
	LaplacianMean        float64 `json:"laplacian_mean"`
	LaplacianStdDev      float64 `json:"laplacian_std_dev"`
	
	// Brightness and luminance
	Brightness           float64 `json:"brightness"`
	AvgLuminance         float64 `json:"average_luminance"`
	LuminanceDistribution [10]float64 `json:"luminance_distribution"` // Histogram bins
	
	// Color metrics
	AvgSaturation        float64    `json:"average_saturation"`
	ChannelBalance       [3]float64 `json:"channel_balance"` // R, G, B
	ChannelMeans         [3]float64 `json:"channel_means"`
	ChannelStdDevs       [3]float64 `json:"channel_std_devs"`
	
	// Exposure metrics
	OverexposedPixelRatio  float64 `json:"overexposed_pixel_ratio"`
	UnderexposedPixelRatio float64 `json:"underexposed_pixel_ratio"`
	DynamicRange           float64 `json:"dynamic_range"`
	
	// Geometric metrics
	SkewAngle            *float64 `json:"skew_angle,omitempty"`
	NumContours          int      `json:"num_contours"`
	EdgePixelRatio       float64  `json:"edge_pixel_ratio"`
	
	// Resolution and size
	Width                int     `json:"width"`
	Height               int     `json:"height"`
	TotalPixels          int     `json:"total_pixels"`
	AspectRatio          float64 `json:"aspect_ratio"`
}

// AppliedThresholds shows all thresholds used in analysis
type AppliedThresholds struct {
	// Sharpness thresholds
	MinLaplacianVariance       float64 `json:"min_laplacian_variance"`
	MaxLaplacianVariance       float64 `json:"max_laplacian_variance"`
	MinLaplacianVarianceForOCR float64 `json:"min_laplacian_variance_for_ocr"`
	
	// Brightness thresholds
	MinBrightness              float64 `json:"min_brightness"`
	MaxBrightness              float64 `json:"max_brightness"`
	
	// Luminance thresholds
	MinLuminance               float64 `json:"min_luminance"`
	MaxLuminance               float64 `json:"max_luminance"`
	
	// Saturation thresholds
	MinSaturation              float64 `json:"min_saturation"`
	
	// Channel balance threshold
	MaxChannelImbalance        float64 `json:"max_channel_imbalance"`
	
	// Exposure thresholds
	OverexposureThreshold      float64 `json:"overexposure_threshold"`
	OversaturationThreshold    float64 `json:"oversaturation_threshold"`
	
	// Geometric thresholds
	MaxSkewAngle               float64 `json:"max_skew_angle"`
	
	// Resolution thresholds
	MinWidth                   int     `json:"min_width"`
	MinHeight                  int     `json:"min_height"`
	MinTotalPixels             int     `json:"min_total_pixels"`
}

// QualityCheckResult represents the result of an individual quality check
type QualityCheckResult struct {
	CheckName       string  `json:"check_name"`
	Passed          bool    `json:"passed"`
	Severity        string  `json:"severity"` // "error", "warning", "info"
	ActualValue     float64 `json:"actual_value"`
	ThresholdValue  float64 `json:"threshold_value"`
	Message         string  `json:"message"`
	Recommendation  string  `json:"recommendation,omitempty"`
	Confidence      float64 `json:"confidence"` // 0-1, confidence in the result
}

// DetailedOCRAnalysis provides comprehensive OCR-specific analysis
type DetailedOCRAnalysis struct {
	// OCR readiness assessment
	OCRReadinessScore   float64 `json:"ocr_readiness_score"` // 0-100
	TextDetectionScore  float64 `json:"text_detection_score"` // 0-100
	
	// Document analysis
	DocumentType        string  `json:"document_type,omitempty"` // "text", "form", "table", etc.
	TextDensity         float64 `json:"text_density"`
	EstimatedTextLines  int     `json:"estimated_text_lines"`
	
	// OCR result (if performed)
	OCRResult           *OCRResult `json:"ocr_result,omitempty"`
	
	// OCR-specific quality issues
	OCRQualityIssues    []string `json:"ocr_quality_issues,omitempty"`
}

// OverallAssessment provides high-level assessment
type OverallAssessment struct {
	QualityGrade        string   `json:"quality_grade"` // "A", "B", "C", "D", "F"
	UsabilityScore      float64  `json:"usability_score"` // 0-100
	RecommendedActions  []string `json:"recommended_actions,omitempty"`
	SuitableFor         []string `json:"suitable_for"` // "web", "print", "ocr", "archive", etc.
	NotSuitableFor      []string `json:"not_suitable_for,omitempty"`
}

// ProcessingDetails provides information about the analysis process
type ProcessingDetails struct {
	AnalysisMode        string            `json:"analysis_mode"` // "basic", "ocr", "quality"
	FeaturesAnalyzed    []string          `json:"features_analyzed"`
	SkippedFeatures     []string          `json:"skipped_features,omitempty"`
	ProcessingOptions   map[string]interface{} `json:"processing_options"`
	PerformanceMetrics  PerformanceMetrics `json:"performance_metrics"`
}

// PerformanceMetrics provides detailed timing information
type PerformanceMetrics struct {
	TotalProcessingTime   float64            `json:"total_processing_time_ms"`
	ImageFetchTime        float64            `json:"image_fetch_time_ms"`
	AnalysisTime          float64            `json:"analysis_time_ms"`
	FeatureTimings        map[string]float64 `json:"feature_timings_ms"`
	MemoryUsage           int64              `json:"memory_usage_bytes,omitempty"`
}

// DetailedAnalysisRequest represents a request for detailed image analysis
type DetailedAnalysisRequest struct {
	URL                 string                 `json:"url" binding:"required,url"`
	AnalysisMode        string                 `json:"analysis_mode,omitempty"` // "basic", "ocr", "comprehensive"
	IncludePerformance  bool                   `json:"include_performance,omitempty"`
	IncludeRawMetrics   bool                   `json:"include_raw_metrics,omitempty"`
	CustomThresholds    *CustomThresholds      `json:"custom_thresholds,omitempty"`
	FeatureFlags        map[string]bool        `json:"feature_flags,omitempty"`
	ExpectedText        string                 `json:"expected_text,omitempty"`
}

// CustomThresholds allows overriding default thresholds
type CustomThresholds struct {
	BlurThreshold         *float64 `json:"blur_threshold,omitempty"`
	OverexposureThreshold *float64 `json:"overexposure_threshold,omitempty"`
	OversaturationThreshold *float64 `json:"oversaturation_threshold,omitempty"`
	MinResolution         *int     `json:"min_resolution,omitempty"`
	MaxSkewAngle          *float64 `json:"max_skew_angle,omitempty"`
}