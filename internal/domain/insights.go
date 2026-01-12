package domain

import "time"

// ChronotypeType represents the user's sleep chronotype classification.
// @Description Chronotype classification based on mid-sleep time.
type ChronotypeType string

const (
	ChronotypeEarlyBird    ChronotypeType = "early_bird"
	ChronotypeIntermediate ChronotypeType = "intermediate"
	ChronotypeNightOwl     ChronotypeType = "night_owl"
	ChronotypeUnknown      ChronotypeType = "unknown"
)

// ChronotypeResult contains the computed chronotype and supporting data.
// @Description Chronotype analysis result.
type ChronotypeResult struct {
	// Chronotype classification
	Chronotype ChronotypeType `json:"chronotype" example:"intermediate"`
	// Mid-sleep time in local timezone (HH:MM format)
	MidSleepLocalTime string `json:"mid_sleep_local_time" example:"03:45"`
	// Minutes after midnight for mid-sleep
	MidSleepMinutesAfterMidnight int `json:"mid_sleep_minutes_after_midnight" example:"225"`
	// Number of days in the analysis window
	WindowDays int `json:"window_days" example:"30"`
	// Number of sleep logs used in calculation
	SleepsUsed int `json:"sleeps_used" example:"28"`
}

// ChronotypeRequest contains query parameters for chronotype endpoint.
type ChronotypeRequest struct {
	WindowDays int `json:"window_days" validate:"omitempty,min=1,max=365"`
	MinSleeps  int `json:"min_sleeps" validate:"omitempty,min=1,max=100"`
}

// DescriptiveStats holds basic statistical measures.
// @Description Basic statistical measures for a metric.
type DescriptiveStats struct {
	Avg float64 `json:"avg" example:"7.2"`
	Std float64 `json:"std" example:"0.8"`
	Min float64 `json:"min" example:"5.5"`
	Max float64 `json:"max" example:"9.0"`
}

// PerSleepMetrics contains per-sleep statistics for a window.
// @Description Per-sleep metrics aggregated over a time window.
type PerSleepMetrics struct {
	// Duration statistics in hours
	Duration DescriptiveStats `json:"duration"`
	// Quality statistics (1-10 scale)
	Quality DescriptiveStats `json:"quality"`
	// Bedtime statistics in minutes after midnight
	Bedtime DescriptiveStats `json:"bedtime"`
	// Number of sleep logs in this window
	SleepCount int `json:"sleep_count" example:"28"`
}

// DailyOverallMetrics contains per-day total sleep statistics.
// @Description Daily total sleep metrics (core + naps combined).
type DailyOverallMetrics struct {
	// Number of days with sleep data
	DaysCount int `json:"days_count" example:"30"`
	// Total daily sleep hours statistics
	TotalDailyHours DescriptiveStats `json:"total_daily_hours"`
	// Target hours for sufficiency calculation
	TargetHours float64 `json:"target_hours" example:"7.0"`
	// Number of days meeting the target
	DaysMeetingTarget int `json:"days_meeting_target" example:"22"`
	// Percentage of days meeting target (0-100)
	DailySufficiencyScore float64 `json:"daily_sufficiency_score" example:"73.3"`
}

// DerivedScores contains computed 0-100 scores.
// @Description Derived scores based on sleep metrics.
type DerivedScores struct {
	// Consistency score based on bedtime variability (0-100)
	ConsistencyScore float64 `json:"consistency_score" example:"75.0"`
	// Sufficiency score based on duration meeting targets (0-100)
	SufficiencyScore float64 `json:"sufficiency_score" example:"80.0"`
	// Overall sleep score combining factors (0-100)
	OverallSleepScore float64 `json:"overall_sleep_score" example:"77.5"`
}

// WindowMetrics contains all metrics for a single time window.
// @Description Complete metrics for a time window.
type WindowMetrics struct {
	// Window start date
	From time.Time `json:"from" example:"2024-01-01T00:00:00Z"`
	// Window end date
	To time.Time `json:"to" example:"2024-01-31T23:59:59Z"`
	// Per-sleep metrics
	PerSleep PerSleepMetrics `json:"per_sleep"`
	// Daily overall metrics (core + naps)
	DailyOverall DailyOverallMetrics `json:"daily_overall"`
	// Derived scores
	Scores DerivedScores `json:"scores"`
}

// MetricsResponse is the response for the metrics endpoint.
// @Description Sleep metrics response with window statistics.
type MetricsResponse struct {
	// Analysis window
	Window struct {
		From time.Time `json:"from" example:"2024-01-01T00:00:00Z"`
		To   time.Time `json:"to" example:"2024-01-31T23:59:59Z"`
	} `json:"window"`
	// Per-sleep metrics
	PerSleep PerSleepMetrics `json:"per_sleep"`
	// Daily overall metrics
	DailyOverall DailyOverallMetrics `json:"daily_overall"`
	// Derived scores
	Scores DerivedScores `json:"scores"`
}

// MetricsRequest contains query parameters for metrics endpoint.
type MetricsRequest struct {
	WindowDays int `json:"window_days" validate:"omitempty,min=1,max=365"`
}

// LLMInsightsOutput contains the structured output from the LLM.
// @Description LLM-generated sleep insights.
type LLMInsightsOutput struct {
	// Summary of sleep patterns (2-3 sentences)
	Summary string `json:"summary" example:"Your sleep has been fairly consistent this week..."`
	// Observations about patterns (3-6 items)
	Observations []string `json:"observations" example:"[\"Average duration of 7.2 hours meets recommended guidelines\"]"`
	// Actionable guidance (3-5 items)
	Guidance []string `json:"guidance" example:"[\"Try to maintain your current bedtime of around 11 PM\"]"`
}

// InsightsContext is the context object sent to the LLM.
// @Description Context data for LLM insights generation.
type InsightsContext struct {
	Chronotype ChronotypeResult `json:"chronotype"`
	History    WindowMetrics    `json:"history"`
	Recent     WindowMetrics    `json:"recent"`
	LastNight  WindowMetrics    `json:"last_night"`
}

// InsightsResponse is the response for the insights endpoint.
// @Description Complete sleep insights response.
type InsightsResponse struct {
	// Chronotype analysis
	Chronotype ChronotypeResult `json:"chronotype"`
	// Metrics for different time windows
	Metrics struct {
		History   WindowMetrics `json:"history"`
		Recent    WindowMetrics `json:"recent"`
		LastNight WindowMetrics `json:"last_night"`
	} `json:"metrics"`
	// LLM-generated insights
	Insights LLMInsightsOutput `json:"insights"`
	// Trace ID for feedback (optional, only present when Langfuse is enabled)
	TraceID string `json:"trace_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}
