// Package tracing - Tail-based sampling for intelligent trace retention
package tracing

import (
	"hash/fnv"
	"math/rand"
	"time"
)

// SamplingConfig configures tail-based sampling strategy
type SamplingConfig struct {
	// Enabled controls whether sampling is active (default: false)
	Enabled bool

	// BaseRate is the default sampling rate for normal traces (0.0 - 1.0)
	// 0.1 = keep 10% of traces, 0.01 = keep 1%
	BaseRate float64

	// AlwaysSampleErrors keeps all traces with errors (default: true)
	AlwaysSampleErrors bool

	// AlwaysSampleSlow keeps all traces exceeding latency threshold (default: true)
	AlwaysSampleSlow bool

	// SlowThresholdMs is latency threshold in milliseconds (default: 5000ms)
	SlowThresholdMs float64

	// AlwaysSampleActionTypes is list of action types to always sample
	// Example: ["DeleteAction", "TransferAction"]
	AlwaysSampleActionTypes []string

	// AlwaysSampleObjectTypes is list of object types to always sample
	// Example: ["Credential", "Database"]
	AlwaysSampleObjectTypes []string

	// AlwaysSampleByStatus keeps all traces with specific status
	// Example: ["failed", "timeout"]
	AlwaysSampleByStatus []string

	// HeadSamplingRate is optional head-based sampling (before seeing result)
	// Applied BEFORE tail-based sampling. 0.0 = disabled (default)
	HeadSamplingRate float64

	// DeterministicSampling uses correlation_id hash for consistent sampling
	// When true, same correlation_id always gets same sampling decision
	// Useful for sampling entire workflows consistently
	DeterministicSampling bool
}

// SamplingDecision contains the result of sampling evaluation
type SamplingDecision struct {
	// ShouldSample indicates whether to keep this trace
	ShouldSample bool

	// Reason explains why this decision was made
	Reason string

	// SamplingRate used for this decision
	SamplingRate float64
}

// Sampler makes tail-based sampling decisions
type Sampler struct {
	config SamplingConfig
	rng    *rand.Rand
}

// NewSampler creates a new tail-based sampler
func NewSampler(config SamplingConfig) *Sampler {
	// Set defaults
	if config.BaseRate == 0 {
		config.BaseRate = 0.1 // 10% by default
	}
	if config.SlowThresholdMs == 0 {
		config.SlowThresholdMs = 5000 // 5 seconds
	}

	// Create RNG with seed based on current time
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	return &Sampler{
		config: config,
		rng:    rng,
	}
}

// ShouldSample makes tail-based sampling decision after seeing trace data
func (s *Sampler) ShouldSample(trace *traceRecord) SamplingDecision {
	// If sampling is disabled, always keep
	if !s.config.Enabled {
		return SamplingDecision{
			ShouldSample: true,
			Reason:       "sampling_disabled",
			SamplingRate: 1.0,
		}
	}

	// Head-based sampling (optional pre-filter)
	if s.config.HeadSamplingRate > 0 {
		if !s.headSample(trace.correlationID, s.config.HeadSamplingRate) {
			return SamplingDecision{
				ShouldSample: false,
				Reason:       "head_sampling_rejected",
				SamplingRate: s.config.HeadSamplingRate,
			}
		}
	}

	// Rule 1: Always sample errors
	if s.config.AlwaysSampleErrors && trace.errorMsg != "" {
		return SamplingDecision{
			ShouldSample: true,
			Reason:       "error_detected",
			SamplingRate: 1.0,
		}
	}

	// Rule 2: Always sample slow traces
	if s.config.AlwaysSampleSlow {
		durationMs := float64(trace.duration) / float64(time.Millisecond)
		if durationMs > s.config.SlowThresholdMs {
			return SamplingDecision{
				ShouldSample: true,
				Reason:       "slow_trace",
				SamplingRate: 1.0,
			}
		}
	}

	// Rule 3: Always sample specific action types
	if s.shouldAlwaysSampleActionType(trace.actionType) {
		return SamplingDecision{
			ShouldSample: true,
			Reason:       "critical_action_type",
			SamplingRate: 1.0,
		}
	}

	// Rule 4: Always sample specific object types
	if s.shouldAlwaysSampleObjectType(trace.objectType) {
		return SamplingDecision{
			ShouldSample: true,
			Reason:       "critical_object_type",
			SamplingRate: 1.0,
		}
	}

	// Rule 5: Always sample specific statuses
	if s.shouldAlwaysSampleStatus(trace.actionStatus) {
		return SamplingDecision{
			ShouldSample: true,
			Reason:       "status_match",
			SamplingRate: 1.0,
		}
	}

	// Rule 6: Base rate sampling for normal traces
	shouldSample := s.probabilitySample(trace.correlationID, s.config.BaseRate)
	reason := "base_rate_sampling"
	if !shouldSample {
		reason = "base_rate_rejected"
	}

	return SamplingDecision{
		ShouldSample: shouldSample,
		Reason:       reason,
		SamplingRate: s.config.BaseRate,
	}
}

// headSample performs head-based sampling (before seeing trace result)
func (s *Sampler) headSample(correlationID string, rate float64) bool {
	return s.probabilitySample(correlationID, rate)
}

// probabilitySample makes sampling decision based on probability
func (s *Sampler) probabilitySample(correlationID string, rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}

	if s.config.DeterministicSampling {
		// Use hash of correlation_id for deterministic sampling
		// Same correlation_id always gets same result
		return s.hashSample(correlationID, rate)
	}

	// Random sampling
	return s.rng.Float64() < rate
}

// hashSample uses consistent hashing for deterministic sampling
func (s *Sampler) hashSample(correlationID string, rate float64) bool {
	h := fnv.New64a()
	h.Write([]byte(correlationID))
	hash := h.Sum64()

	// Map hash to [0, 1) range
	threshold := float64(hash) / float64(^uint64(0))

	return threshold < rate
}

// shouldAlwaysSampleActionType checks if action type should always be sampled
func (s *Sampler) shouldAlwaysSampleActionType(actionType string) bool {
	for _, at := range s.config.AlwaysSampleActionTypes {
		if actionType == at {
			return true
		}
	}
	return false
}

// shouldAlwaysSampleObjectType checks if object type should always be sampled
func (s *Sampler) shouldAlwaysSampleObjectType(objectType string) bool {
	for _, ot := range s.config.AlwaysSampleObjectTypes {
		if objectType == ot {
			return true
		}
	}
	return false
}

// shouldAlwaysSampleStatus checks if status should always be sampled
func (s *Sampler) shouldAlwaysSampleStatus(status string) bool {
	for _, s := range s.config.AlwaysSampleByStatus {
		if status == s {
			return true
		}
	}
	return false
}

// SamplingStats tracks sampling statistics
type SamplingStats struct {
	TotalTraces    int64
	SampledTraces  int64
	RejectedTraces int64
	ErrorTraces    int64
	SlowTraces     int64
	CriticalTraces int64
	BaseRateTraces int64
	HeadRejected   int64
	SamplingRate   float64
	EffectiveRate  float64
}

// UpdateStats updates sampling statistics
func (s *Sampler) UpdateStats(stats *SamplingStats, decision SamplingDecision) {
	stats.TotalTraces++

	if decision.ShouldSample {
		stats.SampledTraces++

		switch decision.Reason {
		case "error_detected":
			stats.ErrorTraces++
		case "slow_trace":
			stats.SlowTraces++
		case "critical_action_type", "critical_object_type", "status_match":
			stats.CriticalTraces++
		case "base_rate_sampling":
			stats.BaseRateTraces++
		}
	} else {
		stats.RejectedTraces++

		if decision.Reason == "head_sampling_rejected" {
			stats.HeadRejected++
		}
	}

	// Calculate effective sampling rate
	if stats.TotalTraces > 0 {
		stats.EffectiveRate = float64(stats.SampledTraces) / float64(stats.TotalTraces)
	}
	stats.SamplingRate = s.config.BaseRate
}

// NewSamplingConfigFromEnv creates sampling config from environment variables
func NewSamplingConfigFromEnv() SamplingConfig {
	config := SamplingConfig{
		AlwaysSampleErrors: true,
		AlwaysSampleSlow:   true,
		SlowThresholdMs:    5000,
		BaseRate:           0.1,
	}

	// Parse from environment
	// SAMPLING_ENABLED=true
	// SAMPLING_BASE_RATE=0.1
	// SAMPLING_SLOW_THRESHOLD_MS=5000
	// SAMPLING_ALWAYS_SAMPLE_ACTIONS=DeleteAction,TransferAction
	// SAMPLING_ALWAYS_SAMPLE_OBJECTS=Credential,Database
	// SAMPLING_DETERMINISTIC=true
	// SAMPLING_HEAD_RATE=0.5

	// TODO: Implement env parsing similar to config.go

	return config
}
