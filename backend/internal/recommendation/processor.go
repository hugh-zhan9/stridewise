package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type Store interface {
	CreateRecommendation(ctx context.Context, rec storage.Recommendation) error
	GetLatestRecommendation(ctx context.Context, userID string) (storage.Recommendation, error)
	CreateRecommendationFeedback(ctx context.Context, feedback storage.RecommendationFeedback) error
	GetUserProfile(ctx context.Context, userID string) (storage.UserProfile, error)
	GetBaselineCurrent(ctx context.Context, userID string) (storage.BaselineCurrent, error)
	CreateWeatherSnapshot(ctx context.Context, s storage.WeatherSnapshot) error
	GetLatestWeatherSnapshot(ctx context.Context, userID string) (storage.WeatherSnapshot, error)
	UpsertWeatherForecasts(ctx context.Context, forecasts []storage.WeatherForecast) error
	GetRecentTrainingSummary(ctx context.Context, userID string, from time.Time, to time.Time) (storage.TrainingLoadSummary, error)
	GetLatestTrainingDiscomfort(ctx context.Context, userID string) (bool, error)
	GetLatestTrainingFeedback(ctx context.Context, userID string) (storage.TrainingFeedback, error)
	GetTrainingSummaryBySource(ctx context.Context, sourceType, sourceID string) (storage.TrainingSummary, error)
}

type Processor struct {
	store       Store
	provider    weather.Provider
	recommender ai.Recommender
	now         func() time.Time
	aiProvider  string
	aiModel     string
}

var ErrFeedbackExists = errors.New("recommendation feedback exists")
var ErrAbilityLevelNotReady = errors.New("ability level not ready")

func NewProcessor(store Store, provider weather.Provider, recommender ai.Recommender) *Processor {
	return &Processor{store: store, provider: provider, recommender: recommender, now: time.Now}
}

func (p *Processor) SetAIInfo(provider, model string) {
	p.aiProvider = provider
	p.aiModel = model
}

func (p *Processor) Generate(ctx context.Context, userID string) (storage.Recommendation, error) {
	if p.store == nil {
		return storage.Recommendation{}, errors.New("recommendation store not configured")
	}
	if p.provider == nil {
		return storage.Recommendation{}, errors.New("weather provider not configured")
	}

	profile, err := p.store.GetUserProfile(ctx, userID)
	if err != nil {
		return storage.Recommendation{}, err
	}
	if profile.AbilityLevel == "" {
		return storage.Recommendation{}, ErrAbilityLevelNotReady
	}
	baseline, err := p.store.GetBaselineCurrent(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return storage.Recommendation{}, err
	}
	if baseline.Status == "" && baseline.DataSessions7d < 3 {
		baseline.Status = "insufficient_data"
	}

	now := p.now()
	location := weather.Location{
		Lat:      profile.LocationLat,
		Lng:      profile.LocationLng,
		Country:  profile.Country,
		Province: profile.Province,
		City:     profile.City,
	}
	weatherInput, weatherRisk, weatherErr := p.fetchWeather(ctx, userID, location)
	forecastInputs := p.fetchForecasts(ctx, userID, location)
	loadSummary, _ := p.store.GetRecentTrainingSummary(ctx, userID, now.Add(-7*24*time.Hour), now)
	hasDiscomfort, _ := p.store.GetLatestTrainingDiscomfort(ctx, userID)
	latestFeedback := p.fetchLatestTrainingFeedback(ctx, userID)

	recoveryStatus := CalcRecoveryStatus(maxFloat(baseline.ACWRSRPE, baseline.ACWRDistance), baseline.Monotony)
	constraints := ai.RecommendationConstraints{
		WeatherRisk:   string(weatherRisk),
		HasDiscomfort: hasDiscomfort,
		HighLoad:      isHighLoad(baseline),
	}

	input := ai.RecommendationInput{
		RequestID: uuid.NewString(),
		UserProfile: ai.RecommendationUserProfile{
			UserID:       profile.UserID,
			AbilityLevel: profile.AbilityLevel,
			GoalType:     profile.GoalType,
			Age:          profile.Age,
			WeightKG:     profile.WeightKG,
			Country:      profile.Country,
			Province:     profile.Province,
			City:         profile.City,
		},
		Baseline: ai.RecommendationBaseline{
			Status:              baseline.Status,
			AcuteLoadSRPE:       baseline.AcuteLoadSRPE,
			ChronicLoadSRPE:     baseline.ChronicLoadSRPE,
			ACWRSRPE:            baseline.ACWRSRPE,
			AcuteLoadDistance:   baseline.AcuteLoadDistance,
			ChronicLoadDistance: baseline.ChronicLoadDistance,
			ACWRDistance:        baseline.ACWRDistance,
			Monotony:            baseline.Monotony,
			Strain:              baseline.Strain,
			PaceAvgSecPerKM:     baseline.PaceAvgSecPerKM,
			PaceLowSecPerKM:     baseline.PaceLowSecPerKM,
			PaceHighSecPerKM:    baseline.PaceHighSecPerKM,
		},
		Weather: ai.RecommendationWeather{
			TemperatureC:      weatherInput.TemperatureC,
			FeelsLikeC:        weatherInput.FeelsLikeC,
			Humidity:          weatherInput.Humidity,
			WindSpeedMS:       weatherInput.WindSpeedMS,
			PrecipitationProb: weatherInput.PrecipitationProb,
			AQI:               weatherInput.AQI,
			UVIndex:           weatherInput.UVIndex,
			RiskLevel:         string(weatherRisk),
			Forecasts:         mapForecasts(forecastInputs),
		},
		TrainingLoad7D: ai.TrainingLoadSummary{
			Sessions: loadSummary.Sessions,
			Distance: loadSummary.Distance,
			Duration: loadSummary.Duration,
		},
		Constraints:            constraints,
		CurrentTime:            now,
		RecoveryStatus:         recoveryStatus,
		LatestTrainingFeedback: latestFeedback,
	}

	output, isFallback := p.callAI(ctx, input, weatherErr)
	if baseline.Status == "insufficient_data" || baseline.DataSessions7d < 3 {
		output = conservativeOutput(profile)
		isFallback = true
	}
	ruleResult := ApplyRules(RuleInput{
		WeatherRisk:    string(weatherRisk),
		HasDiscomfort:  hasDiscomfort,
		HighLoad:       isHighLoad(baseline),
		RecoveryStatus: recoveryStatus,
	}, output)

	if ruleResult.Output.RiskLevel == "" {
		ruleResult.Output.RiskLevel = string(weatherRisk)
	}

	overrideJSON := []byte(`{}`)
	if ruleResult.OverrideReason != "" {
		overrideJSON = mustJSON(map[string]string{"reason": ruleResult.OverrideReason})
	}

	rec := storage.Recommendation{
		RecID:              uuid.NewString(),
		UserID:             userID,
		RecommendationDate: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		CreatedAt:          now.UTC(),
		InputJSON:          mustJSON(input),
		OutputJSON:         mustJSON(ruleResult.Output),
		RiskLevel:          ruleResult.Output.RiskLevel,
		OverrideJSON:       overrideJSON,
		IsFallback:         isFallback,
		AIProvider:         defaultAIProvider(p.aiProvider),
		AIModel:            defaultAIModel(p.aiModel),
		PromptVersion:      "v1",
		EngineVersion:      "v1",
	}
	if err := p.store.CreateRecommendation(ctx, rec); err != nil {
		return storage.Recommendation{}, err
	}
	return rec, nil
}

func maxFloat(a float64, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}

func mapForecasts(input []weather.ForecastInput) []ai.RecommendationForecast {
	if len(input) == 0 {
		return []ai.RecommendationForecast{}
	}
	out := make([]ai.RecommendationForecast, 0, len(input))
	for _, f := range input {
		out = append(out, ai.RecommendationForecast{
			Date:             f.Date.Format("2006-01-02"),
			TempMaxC:         f.TempMaxC,
			TempMinC:         f.TempMinC,
			Humidity:         f.Humidity,
			PrecipMM:         f.PrecipMM,
			PressureHPA:      f.PressureHPA,
			VisibilityKM:     f.VisibilityKM,
			CloudPct:         f.CloudPct,
			UVIndex:          f.UVIndex,
			TextDay:          f.TextDay,
			TextNight:        f.TextNight,
			IconDay:          f.IconDay,
			IconNight:        f.IconNight,
			Wind360Day:       f.Wind360Day,
			WindDirDay:       f.WindDirDay,
			WindScaleDay:     f.WindScaleDay,
			WindSpeedDayMS:   f.WindSpeedDayMS,
			Wind360Night:     f.Wind360Night,
			WindDirNight:     f.WindDirNight,
			WindScaleNight:   f.WindScaleNight,
			WindSpeedNightMS: f.WindSpeedNightMS,
			SunriseTime:      formatTimePtr(f.SunriseTime),
			SunsetTime:       formatTimePtr(f.SunsetTime),
			MoonriseTime:     formatTimePtr(f.MoonriseTime),
			MoonsetTime:      formatTimePtr(f.MoonsetTime),
			MoonPhase:        f.MoonPhase,
			MoonPhaseIcon:    f.MoonPhaseIcon,
		})
	}
	return out
}

func formatTimePtr(input *time.Time) *string {
	if input == nil {
		return nil
	}
	val := input.Format("15:04:05")
	return &val
}

func conservativeOutput(profile storage.UserProfile) RecommendationOutput {
	volume := conservativeTargetVolume(profile.WeeklyDistanceKM)
	shouldRun := true
	riskLevel := "green"
	if strings.ToLower(profile.RecentDiscomfort) == "yes" {
		shouldRun = false
		riskLevel = "red"
	}
	return RecommendationOutput{
		ShouldRun:           shouldRun,
		WorkoutType:         "easy_run",
		IntensityRange:      "低强度",
		TargetVolume:        volume,
		SuggestedTimeWindow: "any",
		RiskLevel:           riskLevel,
		HydrationTip:        "",
		ClothingTip:         "",
		Explanation:         []string{"问卷默认保守模板：当前训练数据不足，建议以低风险方式开始。"},
		AlternativeWorkouts: []AlternativeWorkout{},
	}
}

func conservativeTargetVolume(weeklyDistance string) string {
	min := weeklyDistanceLowerBound(weeklyDistance)
	if min <= 0 {
		return "0 km"
	}
	minTarget := min * 0.2
	maxTarget := min * 0.3
	return fmt.Sprintf("%.1f-%.1f km", minTarget, maxTarget)
}

func weeklyDistanceLowerBound(value string) float64 {
	switch value {
	case "0-5":
		return 0
	case "5-15":
		return 5
	case "15-30":
		return 15
	case "30+":
		return 30
	default:
		return 0
	}
}

func (p *Processor) GetLatest(ctx context.Context, userID string) (storage.Recommendation, error) {
	return p.store.GetLatestRecommendation(ctx, userID)
}

func (p *Processor) Feedback(ctx context.Context, recID string, userID string, useful string, reason string) error {
	if useful != "yes" && useful != "neutral" && useful != "no" {
		return errors.New("useful invalid")
	}
	feedback := storage.RecommendationFeedback{
		FeedbackID: uuid.NewString(),
		RecID:      recID,
		UserID:     userID,
		Useful:     useful,
		Reason:     reason,
	}
	if err := p.store.CreateRecommendationFeedback(ctx, feedback); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrFeedbackExists
		}
		return err
	}
	return nil
}

func (p *Processor) callAI(ctx context.Context, input ai.RecommendationInput, weatherErr error) (RecommendationOutput, bool) {
	if weatherErr != nil {
		return fallbackOutput(), true
	}
	if p.recommender == nil {
		return fallbackOutput(), true
	}
	out, err := p.recommender.Recommend(ctx, input)
	if err != nil {
		return fallbackOutput(), true
	}
	return convertOutput(out), false
}

func (p *Processor) fetchWeather(ctx context.Context, userID string, location weather.Location) (weather.SnapshotInput, weather.RiskLevel, error) {
	input, err := p.provider.GetSnapshot(ctx, location)
	if err == nil {
		risk := weather.ClassifyRisk(input)
		_ = p.store.CreateWeatherSnapshot(ctx, storage.WeatherSnapshot{
			UserID:            userID,
			Date:              time.Now().UTC().Truncate(24 * time.Hour),
			TemperatureC:      input.TemperatureC,
			FeelsLikeC:        input.FeelsLikeC,
			Humidity:          input.Humidity,
			WindSpeedMS:       input.WindSpeedMS,
			PrecipitationProb: input.PrecipitationProb,
			AQI:               input.AQI,
			UVIndex:           input.UVIndex,
			RiskLevel:         string(risk),
		})
		return input, risk, nil
	}
	snap, snapErr := p.store.GetLatestWeatherSnapshot(ctx, userID)
	if snapErr != nil {
		return weather.SnapshotInput{}, weather.RiskRed, err
	}
	input = weather.SnapshotInput{
		TemperatureC:      snap.TemperatureC,
		FeelsLikeC:        snap.FeelsLikeC,
		Humidity:          snap.Humidity,
		WindSpeedMS:       snap.WindSpeedMS,
		PrecipitationProb: snap.PrecipitationProb,
		AQI:               snap.AQI,
		UVIndex:           snap.UVIndex,
	}
	risk := weather.ClassifyRisk(input)
	return input, risk, nil
}

func (p *Processor) fetchForecasts(ctx context.Context, userID string, location weather.Location) []weather.ForecastInput {
	forecasts, err := p.provider.GetForecast(ctx, location)
	if err != nil {
		return nil
	}
	if len(forecasts) == 0 {
		return nil
	}
	storageForecasts := make([]storage.WeatherForecast, 0, len(forecasts))
	for _, f := range forecasts {
		storageForecasts = append(storageForecasts, storage.WeatherForecast{
			ForecastID:       uuid.NewString(),
			UserID:           userID,
			ForecastDate:     f.Date,
			TempMaxC:         f.TempMaxC,
			TempMinC:         f.TempMinC,
			Humidity:         f.Humidity,
			PrecipMM:         f.PrecipMM,
			PressureHPA:      f.PressureHPA,
			VisibilityKM:     f.VisibilityKM,
			CloudPct:         f.CloudPct,
			UVIndex:          f.UVIndex,
			TextDay:          f.TextDay,
			TextNight:        f.TextNight,
			IconDay:          f.IconDay,
			IconNight:        f.IconNight,
			Wind360Day:       f.Wind360Day,
			WindDirDay:       f.WindDirDay,
			WindScaleDay:     f.WindScaleDay,
			WindSpeedDayMS:   f.WindSpeedDayMS,
			Wind360Night:     f.Wind360Night,
			WindDirNight:     f.WindDirNight,
			WindScaleNight:   f.WindScaleNight,
			WindSpeedNightMS: f.WindSpeedNightMS,
			SunriseTime:      f.SunriseTime,
			SunsetTime:       f.SunsetTime,
			MoonriseTime:     f.MoonriseTime,
			MoonsetTime:      f.MoonsetTime,
			MoonPhase:        f.MoonPhase,
			MoonPhaseIcon:    f.MoonPhaseIcon,
		})
	}
	_ = p.store.UpsertWeatherForecasts(ctx, storageForecasts)
	return forecasts
}

func (p *Processor) fetchLatestTrainingFeedback(ctx context.Context, userID string) *ai.RecommendationTrainingFeedback {
	if p.store == nil {
		return nil
	}
	feedback, err := p.store.GetLatestTrainingFeedback(ctx, userID)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(feedback.Content) == "" {
		return nil
	}
	summary, err := p.store.GetTrainingSummaryBySource(ctx, feedback.SourceType, feedback.SourceID)
	summaryInput := &ai.RecommendationTrainingSummary{}
	if err == nil {
		summaryInput.CompletionRate = summary.CompletionRate
		summaryInput.IntensityMatch = summary.IntensityMatch
		summaryInput.RecoveryAdvice = summary.RecoveryAdvice
		summaryInput.AnomalyNotes = summary.AnomalyNotes
		summaryInput.PerformanceNotes = summary.PerformanceNotes
		summaryInput.NextSuggestion = summary.NextSuggestion
	}
	createdAt := ""
	if !feedback.CreatedAt.IsZero() {
		createdAt = feedback.CreatedAt.UTC().Format(time.RFC3339)
	}
	return &ai.RecommendationTrainingFeedback{
		SourceType: feedback.SourceType,
		SourceID:   feedback.SourceID,
		CreatedAt:  createdAt,
		Content:    feedback.Content,
		Summary:    summaryInput,
	}
}

func isHighLoad(b storage.BaselineCurrent) bool {
	if b.ACWRSRPE > 1.5 || b.ACWRDistance > 1.5 {
		return true
	}
	return false
}

func fallbackOutput() RecommendationOutput {
	return RecommendationOutput{
		ShouldRun:           false,
		WorkoutType:         "rest",
		IntensityRange:      "low",
		TargetVolume:        "0",
		SuggestedTimeWindow: "any",
		RiskLevel:           "red",
		HydrationTip:        "保持补水",
		ClothingTip:         "注意保暖",
		Explanation:         []string{"AI 不可用，采用保守建议", "安全优先建议休息"},
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

func convertOutput(out ai.RecommendationOutput) RecommendationOutput {
	alts := make([]AlternativeWorkout, 0, len(out.AlternativeWorkouts))
	for _, a := range out.AlternativeWorkouts {
		alts = append(alts, AlternativeWorkout{
			Type:        a.Type,
			Title:       a.Title,
			DurationMin: a.DurationMin,
			Intensity:   a.Intensity,
			Tips:        a.Tips,
		})
	}
	return RecommendationOutput{
		ShouldRun:           out.ShouldRun,
		WorkoutType:         out.WorkoutType,
		IntensityRange:      out.IntensityRange,
		TargetVolume:        out.TargetVolume,
		SuggestedTimeWindow: out.SuggestedTimeWindow,
		RiskLevel:           out.RiskLevel,
		HydrationTip:        out.HydrationTip,
		ClothingTip:         out.ClothingTip,
		Explanation:         out.Explanation,
		AlternativeWorkouts: alts,
	}
}

func defaultAIProvider(provider string) string {
	if provider == "" {
		return "openai"
	}
	return provider
}

func defaultAIModel(model string) string {
	if model == "" {
		return "unknown"
	}
	return model
}
