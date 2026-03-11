package recommendation

type RuleInput struct {
	WeatherRisk   string
	HasDiscomfort bool
	HighLoad      bool
	RecoveryStatus string
}

type RuleResult struct {
	Output         RecommendationOutput
	OverrideReason string
}

func ApplyRules(input RuleInput, output RecommendationOutput) RuleResult {
	override := ""
	if input.HasDiscomfort {
		override = "user_discomfort"
	} else if input.RecoveryStatus == "red" {
		override = "recovery_red"
	} else if input.WeatherRisk == "red" {
		override = "weather_red"
	} else if input.HighLoad {
		override = "high_load"
	}
	if override != "" {
		output.ShouldRun = false
		if output.WorkoutType == "" {
			output.WorkoutType = "rest"
		}
		if len(output.AlternativeWorkouts) == 0 {
			output.AlternativeWorkouts = []AlternativeWorkout{
				{Type: "treadmill", Title: "室内跑步机轻松跑", DurationMin: 30, Intensity: "low"},
				{Type: "strength", Title: "基础力量训练", DurationMin: 20, Intensity: "low"},
				{Type: "mobility", Title: "拉伸与灵活性恢复", DurationMin: 15, Intensity: "low"},
			}
		}
		if output.RiskLevel == "" {
			output.RiskLevel = "red"
		}
		if len(output.Explanation) < 2 {
			output.Explanation = []string{"安全优先，建议休息", "触发安全规则降级"}
		}
	}
	return RuleResult{Output: output, OverrideReason: override}
}
