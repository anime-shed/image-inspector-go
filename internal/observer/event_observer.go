package observer

import (
	"context"
	"sync"
	"time"
	
	"github.com/sirupsen/logrus"
)

// AnalysisEvent represents an analysis event
type AnalysisEvent struct {
	EventType     EventType   `json:"event_type"`
	Timestamp     time.Time   `json:"timestamp"`
	ImageURL      string      `json:"image_url"`
	ProcessingTime time.Duration `json:"processing_time"`
	Success       bool        `json:"success"`
	ErrorMessage  string      `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EventType represents the type of analysis event
type EventType string

const (
	// AnalysisStarted when analysis begins
	AnalysisStarted EventType = "analysis_started"
	// AnalysisCompleted when analysis finishes successfully
	AnalysisCompleted EventType = "analysis_completed"
	// AnalysisFailed when analysis fails
	AnalysisFailed EventType = "analysis_failed"
	// ImageFetched when image is successfully fetched
	ImageFetched EventType = "image_fetched"
	// ImageFetchFailed when image fetch fails
	ImageFetchFailed EventType = "image_fetch_failed"
)

// Observer defines the interface for event observers
type Observer interface {
	OnEvent(ctx context.Context, event AnalysisEvent)
	GetObserverName() string
}

// Subject defines the interface for event publishers
type Subject interface {
	Subscribe(observer Observer)
	Unsubscribe(observer Observer)
	NotifyObservers(ctx context.Context, event AnalysisEvent)
}

// LoggingObserver logs analysis events
type LoggingObserver struct {
	logger *logrus.Logger
}

// NewLoggingObserver creates a new logging observer
func NewLoggingObserver(logger *logrus.Logger) Observer {
	return &LoggingObserver{
		logger: logger,
	}
}

// OnEvent handles analysis events by logging them
func (o *LoggingObserver) OnEvent(ctx context.Context, event AnalysisEvent) {
	fields := logrus.Fields{
		"event_type":      event.EventType,
		"image_url":       event.ImageURL,
		"processing_time": event.ProcessingTime,
		"success":         event.Success,
	}
	
	if event.ErrorMessage != "" {
		fields["error"] = event.ErrorMessage
	}
	
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			fields[k] = v
		}
	}
	
	switch event.EventType {
	case AnalysisStarted:
		o.logger.WithFields(fields).Info("Image analysis started")
	case AnalysisCompleted:
		o.logger.WithFields(fields).Info("Image analysis completed")
	case AnalysisFailed:
		o.logger.WithFields(fields).Error("Image analysis failed")
	case ImageFetched:
		o.logger.WithFields(fields).Debug("Image fetched successfully")
	case ImageFetchFailed:
		o.logger.WithFields(fields).Error("Image fetch failed")
	default:
		o.logger.WithFields(fields).Info("Analysis event occurred")
	}
}

// GetObserverName returns the observer name
func (o *LoggingObserver) GetObserverName() string {
	return "logging_observer"
}

// MetricsObserver collects metrics from analysis events
type MetricsObserver struct {
	mu                sync.RWMutex
	totalAnalyses     int64
	successfulAnalyses int64
	failedAnalyses    int64
	totalProcessingTime time.Duration
}

// NewMetricsObserver creates a new metrics observer
func NewMetricsObserver() Observer {
	return &MetricsObserver{}
}

// OnEvent handles analysis events by collecting metrics
func (o *MetricsObserver) OnEvent(ctx context.Context, event AnalysisEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	switch event.EventType {
	case AnalysisStarted:
		o.totalAnalyses++
	case AnalysisCompleted:
		o.successfulAnalyses++
		o.totalProcessingTime += event.ProcessingTime
	case AnalysisFailed:
		o.failedAnalyses++
	}
}

// GetObserverName returns the observer name
func (o *MetricsObserver) GetObserverName() string {
	return "metrics_observer"
}

// GetMetrics returns current metrics
func (o *MetricsObserver) GetMetrics() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	avgProcessingTime := time.Duration(0)
	if o.successfulAnalyses > 0 {
		avgProcessingTime = o.totalProcessingTime / time.Duration(o.successfulAnalyses)
	}
	
	return map[string]interface{}{
		"total_analyses":        o.totalAnalyses,
		"successful_analyses":   o.successfulAnalyses,
		"failed_analyses":       o.failedAnalyses,
		"total_processing_time": o.totalProcessingTime,
		"avg_processing_time":   avgProcessingTime,
	}
}

// EventPublisher implements the Subject interface
type EventPublisher struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher() Subject {
	return &EventPublisher{
		observers: make([]Observer, 0),
	}
}

// Subscribe adds an observer
func (p *EventPublisher) Subscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// Unsubscribe removes an observer
func (p *EventPublisher) Unsubscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for i, obs := range p.observers {
		if obs.GetObserverName() == observer.GetObserverName() {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
			break
		}
	}
}

// NotifyObservers notifies all observers of an event
func (p *EventPublisher) NotifyObservers(ctx context.Context, event AnalysisEvent) {
	p.mu.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()
	
	// Notify observers concurrently
	for _, observer := range observers {
		go func(obs Observer) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash the application
					logrus.WithField("observer", obs.GetObserverName()).
						WithField("panic", r).
						Error("Observer panicked while handling event")
				}
			}()
			obs.OnEvent(ctx, event)
		}(observer)
	}
}