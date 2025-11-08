# Tail-Based Sampling Strategy

Intelligent trace sampling that reduces storage costs while maintaining full visibility into errors, slow operations, and critical actions.

## Overview

Tail-based sampling makes the decision whether to keep a trace **after** seeing the trace data, unlike head-based sampling which decides at the start. This allows us to:

- **Keep 100% of problematic traces**: Errors, slow operations, critical actions
- **Sample normal traffic**: Only keep a percentage of fast, successful traces
- **Reduce storage costs**: By 50-90% depending on error rate and traffic patterns
- **Maintain full debugging capability**: Never miss important trace data

## How It Works

```
Request → Execute → Collect Trace Data → Sampling Decision → Export or Drop
                                              ↓
                                    ┌─────────┴──────────┐
                                    │                    │
                              Is it problematic?    Is it normal?
                                    │                    │
                                   Yes                   No
                                    │                    │
                                Keep 100%          Sample at base rate
```

## Configuration

### Basic Setup

```go
import "eve.evalgo.org/tracing"

// Create tracer with sampling enabled
tracer := tracing.New(tracing.Config{
	ServiceID:       "containerservice",
	DB:              db,
	S3Client:        s3Client,
	SamplingEnabled: true,
	SamplingConfig: tracing.SamplingConfig{
		Enabled:             true,
		BaseRate:            0.1,  // Keep 10% of normal traces
		AlwaysSampleErrors:  true, // Keep all errors
		AlwaysSampleSlow:    true, // Keep all slow traces
		SlowThresholdMs:     5000, // 5 seconds
	},
})
```

### Advanced Configuration

```go
samplingConfig := tracing.SamplingConfig{
	// Enable sampling
	Enabled: true,

	// Base sampling rate for normal, fast, successful traces
	BaseRate: 0.05, // Keep 5% of normal traffic

	// Error sampling
	AlwaysSampleErrors: true, // Keep all traces with errors

	// Latency sampling
	AlwaysSampleSlow:    true,  // Keep all slow traces
	SlowThresholdMs:     3000,  // 3 seconds threshold

	// Action type sampling - always keep these action types
	AlwaysSampleActionTypes: []string{
		"DeleteAction",      // Deletions are critical
		"TransferAction",    // Data transfers important
		"UpdateAction",      // Updates worth tracking
	},

	// Object type sampling - always keep these object types
	AlwaysSampleObjectTypes: []string{
		"Credential",        // Security-related
		"Database",          // Database operations
		"DataFeed",          // Data pipelines
	},

	// Status sampling - always keep these statuses
	AlwaysSampleByStatus: []string{
		"failed",
		"timeout",
		"unauthorized",
	},

	// Head sampling (optional pre-filter)
	HeadSamplingRate: 0.5, // Drop 50% before even looking at result

	// Deterministic sampling - same correlation_id always gets same decision
	DeterministicSampling: true,
}

tracer := tracing.New(tracing.Config{
	ServiceID:       "containerservice",
	SamplingEnabled: true,
	SamplingConfig:  samplingConfig,
	// ... other config
})
```

## Sampling Rules Priority

Sampling decisions are evaluated in this order:

1. **Sampling disabled** → Keep all traces
2. **Head sampling** (if enabled) → Pre-filter before evaluation
3. **Error detected** → Keep (if AlwaysSampleErrors=true)
4. **Slow trace** → Keep (if AlwaysSampleSlow=true and duration > threshold)
5. **Critical action type** → Keep (if in AlwaysSampleActionTypes)
6. **Critical object type** → Keep (if in AlwaysSampleObjectTypes)
7. **Status match** → Keep (if in AlwaysSampleByStatus)
8. **Base rate sampling** → Probabilistic decision using BaseRate

## Environment Variables

```bash
# Enable sampling
export SAMPLING_ENABLED=true

# Base sampling rate (0.0 to 1.0)
export SAMPLING_BASE_RATE=0.1

# Slow threshold in milliseconds
export SAMPLING_SLOW_THRESHOLD_MS=5000

# Always sample these action types (comma-separated)
export SAMPLING_ALWAYS_SAMPLE_ACTIONS="DeleteAction,TransferAction,UpdateAction"

# Always sample these object types (comma-separated)
export SAMPLING_ALWAYS_SAMPLE_OBJECTS="Credential,Database,DataFeed"

# Deterministic sampling (true/false)
export SAMPLING_DETERMINISTIC=true

# Head sampling rate (optional, 0.0 to 1.0)
export SAMPLING_HEAD_RATE=0.5
```

## Sampling Strategies

### 1. Conservative (Default)
**Goal**: Catch all problems, moderate storage reduction

```go
BaseRate:           0.1,  // 10% of normal traffic
AlwaysSampleErrors: true,
AlwaysSampleSlow:   true,
SlowThresholdMs:    5000, // 5s
```

**Expected reduction**: ~50% storage (assuming 5% error rate, 5% slow traces)

### 2. Aggressive
**Goal**: Maximum storage reduction, acceptable for mature systems

```go
BaseRate:           0.01, // 1% of normal traffic
AlwaysSampleErrors: true,
AlwaysSampleSlow:   true,
SlowThresholdMs:    10000, // 10s
```

**Expected reduction**: ~90% storage (assuming low error/slow rate)

### 3. Business-Critical Only
**Goal**: Only keep business-critical operations

```go
BaseRate: 0,  // Drop all normal traffic
AlwaysSampleActionTypes: []string{
	"DeleteAction",
	"TransferAction",
	"UpdateAction",
},
AlwaysSampleObjectTypes: []string{
	"Credential",
	"Database",
},
```

### 4. Debugging Mode (Temporarily)
**Goal**: Full visibility during incident investigation

```go
BaseRate:           1.0,  // Keep everything
AlwaysSampleErrors: true,
AlwaysSampleSlow:   true,
```

## Deterministic vs Random Sampling

### Deterministic Sampling (Recommended)

Uses consistent hashing of `correlation_id` so the same workflow always gets the same sampling decision.

**Advantages**:
- Entire workflow sampled consistently (all or nothing)
- Reproducible sampling for testing
- Better for multi-service workflows

```go
DeterministicSampling: true
```

**How it works**:
```go
hash(correlation_id) % 100 < (base_rate * 100)
```

### Random Sampling

Each trace gets an independent random decision.

**Advantages**:
- Truly random distribution
- Better statistical properties for very high volume

```go
DeterministicSampling: false
```

## Head + Tail Sampling

Combine head-based and tail-based sampling for maximum efficiency:

```go
HeadSamplingRate: 0.5,  // Drop 50% immediately (before even looking)
BaseRate:         0.2,  // Of the remaining 50%, keep 20% of normal traces
```

**Effective rate**: 0.5 × 0.2 = 0.1 (10% of normal traffic)

**Use case**: Very high traffic volume where even tail-based sampling is expensive

## Monitoring Sampling

### Prometheus Metrics

```promql
# Sampling decision rate
rate(eve_tracing_sampling_decisions_total[5m])

# Sampling rejection rate
rate(eve_tracing_sampling_decisions_total{decision="rejected"}[5m])

# Sampling by reason
sum(rate(eve_tracing_sampling_decisions_total[5m])) by (reason)

# Effective sampling rate
sum(rate(eve_tracing_sampling_decisions_total{decision="sampled"}[5m])) /
sum(rate(eve_tracing_sampling_decisions_total[5m]))
```

### Dashboard Panels

```yaml
# Effective Sampling Rate
expr: |
  sum(rate(eve_tracing_sampling_decisions_total{decision="sampled"}[5m])) /
  sum(rate(eve_tracing_sampling_decisions_total[5m]))

# Sampling Breakdown by Reason
expr: |
  sum(rate(eve_tracing_sampling_decisions_total{decision="sampled"}[5m])) by (reason)
```

## Examples

### Example 1: E-commerce Site

**Requirements**:
- Track all checkout errors (critical business impact)
- Track slow checkouts (>3s)
- Sample normal browsing at 5%

```go
SamplingConfig{
	Enabled:             true,
	BaseRate:            0.05,
	AlwaysSampleErrors:  true,
	AlwaysSampleSlow:    true,
	SlowThresholdMs:     3000,
	AlwaysSampleActionTypes: []string{
		"PaymentAction",    // All payments
		"CreateAction",     // All order creations
	},
}
```

### Example 2: Data Pipeline

**Requirements**:
- Track all data transfer errors
- Track all slow transfers (>10s)
- Track all deletions
- Sample normal data reads at 1%

```go
SamplingConfig{
	Enabled:             true,
	BaseRate:            0.01,
	AlwaysSampleErrors:  true,
	AlwaysSampleSlow:    true,
	SlowThresholdMs:     10000,
	AlwaysSampleActionTypes: []string{
		"TransferAction",
		"DeleteAction",
	},
	AlwaysSampleObjectTypes: []string{
		"DataFeed",
		"Database",
	},
}
```

### Example 3: Microservices with Workflows

**Requirements**:
- Consistent sampling across multi-service workflows
- Keep all errors
- 10% sampling rate

```go
SamplingConfig{
	Enabled:               true,
	BaseRate:              0.1,
	AlwaysSampleErrors:    true,
	DeterministicSampling: true,  // Key: ensures entire workflow sampled together
}
```

## Cost Analysis

### Storage Savings

Assume:
- 1M traces/day
- 5% error rate
- 5% slow trace rate
- 90% normal traffic

**Without sampling**:
- Stored: 1M traces/day

**With tail-based sampling (BaseRate=0.1)**:
- Errors: 50K traces (5% × 1M, kept 100%)
- Slow: 50K traces (5% × 1M, kept 100%)
- Normal: 90K traces (90% × 1M × 0.1 sampling rate)
- **Total stored: 190K traces/day (81% reduction)**

**Cost savings** (at $0.10/GB):
- Trace size: ~10KB average
- Original cost: 1M × 10KB × $0.10/GB = $1.00/day
- New cost: 190K × 10KB × $0.10/GB = $0.19/day
- **Savings: $0.81/day ($296/year)**

### Query Performance

Sampling also reduces:
- Database query times (fewer rows)
- S3 listing times (fewer objects)
- Dashboard load times
- MCP tool response times

## Best Practices

### DO

1. **Start conservative**: Begin with 10% base rate, adjust down
2. **Always sample errors**: Never lose debugging data
3. **Always sample slow traces**: Catch performance regressions
4. **Use deterministic sampling for workflows**: Keeps multi-service traces together
5. **Monitor sampling metrics**: Track effective sampling rate
6. **Sample critical actions**: DeleteAction, payment operations, etc.
7. **Review sampling quarterly**: Adjust based on traffic patterns

### DON'T

1. **Don't set BaseRate too low initially**: Start at 10%, not 1%
2. **Don't disable error sampling**: You'll lose critical debug data
3. **Don't sample security events**: Credentials, auth failures, etc.
4. **Don't forget to monitor**: Track sampling decisions in Prometheus
5. **Don't use aggressive sampling in new systems**: Need baseline first
6. **Don't sample during incidents**: Temporarily disable sampling

## Troubleshooting

### Too Few Traces

**Problem**: Not enough trace data for debugging

**Solution**:
```go
// Increase base rate
BaseRate: 0.2  // from 0.1

// Lower slow threshold
SlowThresholdMs: 3000  // from 5000

// Add more critical action types
AlwaysSampleActionTypes: []string{
	"CreateAction",
	"UpdateAction",
	"DeleteAction",
}
```

### Too Many Traces

**Problem**: Still storing too much data

**Solution**:
```go
// Decrease base rate
BaseRate: 0.05  // from 0.1

// Add head sampling
HeadSamplingRate: 0.5  // Pre-filter 50%

// Increase slow threshold
SlowThresholdMs: 10000  // from 5000
```

### Inconsistent Workflow Traces

**Problem**: Some services have traces, others don't for same workflow

**Solution**:
```go
// Enable deterministic sampling
DeterministicSampling: true
```

### Missing Important Traces

**Problem**: Critical operations being sampled out

**Solution**:
```go
// Add to AlwaysSampleActionTypes
AlwaysSampleActionTypes: []string{
	"YourCriticalAction",
}

// Or add to AlwaysSampleObjectTypes
AlwaysSampleObjectTypes: []string{
	"YourCriticalObject",
}
```

## Related Documentation

- Async Export: `/home/opunix/eve/tracing/async.go`
- Metrics: `/home/opunix/eve/tracing/metrics.go`
- Configuration: `/home/opunix/eve/tracing/config.go`
- Grafana Dashboards: `/home/opunix/eve/grafana/dashboards/`
