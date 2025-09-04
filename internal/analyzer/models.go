package analyzer

import (
	"github.com/anime-shed/image-inspector-go/pkg/models"
)

// AnalysisResult is now an alias to the shared models.AnalysisResult
// This maintains backward compatibility while using the shared model
type AnalysisResult = models.AnalysisResult

// metrics holds internal calculation results
type metrics struct {
	avgLuminance, avgSaturation float64
	avgR, avgG, avgB            float64
}