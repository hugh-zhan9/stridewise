package recommendation

type RecommendationOutput struct {
	ShouldRun           bool
	WorkoutType         string
	IntensityRange      string
	TargetVolume        string
	SuggestedTimeWindow string
	RiskLevel           string
	HydrationTip        string
	ClothingTip         string
	Explanation         []string
	AlternativeWorkouts []AlternativeWorkout
}

type AlternativeWorkout struct {
	Type        string
	Title       string
	DurationMin int
	Intensity   string
	Tips        []string
}
