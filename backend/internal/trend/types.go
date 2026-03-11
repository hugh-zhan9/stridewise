package trend

type TrendSummary struct {
	Sessions             int            `json:"sessions"`
	DistanceKM           float64        `json:"distance_km"`
	DurationSec          int            `json:"duration_sec"`
	AvgPaceSecPerKM      int            `json:"avg_pace_sec_per_km"`
	AvgRPE               float64        `json:"avg_rpe"`
	SummaryCount         int            `json:"summary_count"`
	CompletionRateDist   map[string]int `json:"completion_rate_dist"`
	IntensityMatchDist   map[string]int `json:"intensity_match_dist"`
	RecoveryAdviceTags   map[string]int `json:"recovery_advice_tags"`
	ACWRSRPE             *float64       `json:"acwr_srpe"`
	ACWRDistance         *float64       `json:"acwr_distance"`
	Monotony             *float64       `json:"monotony"`
	Strain               *float64       `json:"strain"`
}

type TrendPoint struct {
	Date            string  `json:"date"`
	Sessions        int     `json:"sessions"`
	DistanceKM      float64 `json:"distance_km"`
	DurationSec     int     `json:"duration_sec"`
	AvgPaceSecPerKM int     `json:"avg_pace_sec_per_km"`
	AvgRPE          float64 `json:"avg_rpe"`
}

type TrendResult struct {
	WindowStart string       `json:"window_start"`
	WindowEnd   string       `json:"window_end"`
	Summary     TrendSummary `json:"summary"`
	Series      []TrendPoint `json:"series"`
}
