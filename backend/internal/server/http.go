package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/middleware"
	"stridewise/backend/internal/recommendation"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
	"stridewise/backend/internal/training"
	"stridewise/backend/internal/weather"
)

type SyncJobCreator interface {
	CreateSyncJob(ctx context.Context, jobID, userID, source string) error
}

type SyncJob struct {
	JobID        string `json:"job_id"`
	UserID       string `json:"user_id"`
	Source       string `json:"source"`
	Status       string `json:"status"`
	RetryCount   int    `json:"retry_count"`
	ErrorMessage string `json:"error_message"`
}

type SyncJobReader interface {
	GetSyncJob(ctx context.Context, jobID string) (SyncJob, error)
}

type SyncJobRetryer interface {
	RetrySyncJob(ctx context.Context, jobID string) (SyncJob, error)
}

type UserProfileStore interface {
	UpsertUserProfile(ctx context.Context, p storage.UserProfile) error
	GetUserProfile(ctx context.Context, userID string) (storage.UserProfile, error)
}

type WeatherStore interface {
	CreateWeatherSnapshot(ctx context.Context, w storage.WeatherSnapshot) error
	GetWeatherSnapshot(ctx context.Context, userID string, date time.Time) (storage.WeatherSnapshot, error)
}

type WeatherProvider interface {
	GetSnapshot(ctx context.Context, location weather.Location) (weather.SnapshotInput, error)
}

type TrainingLogStore interface {
	HasTrainingConflict(ctx context.Context, userID string, start time.Time, end time.Time, excludeLogID string) (bool, error)
	CreateTrainingLog(ctx context.Context, log storage.TrainingLog) error
	UpdateTrainingLog(ctx context.Context, log storage.TrainingLog) error
	SoftDeleteTrainingLog(ctx context.Context, logID string) error
	ListTrainingLogs(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingLog, error)
	GetTrainingLog(ctx context.Context, logID string) (storage.TrainingLog, error)
}

type AsyncJobStore interface {
	CreateAsyncJob(ctx context.Context, job storage.AsyncJob) error
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
}

type BaselineStore interface {
	GetBaselineCurrent(ctx context.Context, userID string) (storage.BaselineCurrent, error)
	ListBaselineHistory(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.BaselineHistory, error)
	ListTrainingSummaries(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingSummary, error)
	CreateTrainingFeedback(ctx context.Context, feedback storage.TrainingFeedback) error
}

type RecommendationService interface {
	Generate(ctx context.Context, userID string) (storage.Recommendation, error)
	GetLatest(ctx context.Context, userID string) (storage.Recommendation, error)
	Feedback(ctx context.Context, recID string, userID string, useful string, reason string) error
}

type createSyncJobRequest struct {
	UserID string `json:"user_id"`
	Source string `json:"source"`
}

type userProfileRequest struct {
	UserID         string   `json:"user_id"`
	Gender         string   `json:"gender"`
	Age            int      `json:"age"`
	HeightCM       int      `json:"height_cm"`
	WeightKG       int      `json:"weight_kg"`
	GoalType       string   `json:"goal_type"`
	GoalCycle      string   `json:"goal_cycle"`
	GoalFrequency  int      `json:"goal_frequency"`
	GoalPace       string   `json:"goal_pace"`
	RunningYears   string   `json:"running_years"`
	WeeklySessions string   `json:"weekly_sessions"`
	WeeklyDistanceKM string `json:"weekly_distance_km"`
	LongestRunKM   string   `json:"longest_run_km"`
	RecentDiscomfort string `json:"recent_discomfort"`
	LocationLat    *float64 `json:"location_lat"`
	LocationLng    *float64 `json:"location_lng"`
	Country        string   `json:"country"`
	Province       string   `json:"province"`
	City           string   `json:"city"`
	LocationSource string   `json:"location_source"`
}

type weatherSnapshotRequest struct {
	UserID string `json:"user_id"`
	Date   string `json:"date"`
}

type trainingLogRequest struct {
	UserID             string  `json:"user_id"`
	TrainingType       string  `json:"training_type"`
	TrainingTypeCustom string  `json:"training_type_custom"`
	StartTime          string  `json:"start_time"`
	Duration           string  `json:"duration"`
	DistanceKM         float64 `json:"distance_km"`
	Pace               string  `json:"pace"`
	RPE                int     `json:"rpe"`
	Discomfort         bool    `json:"discomfort"`
}

type trainingFeedbackRequest struct {
	UserID     string `json:"user_id"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Content    string `json:"content"`
}

type trainingSummaryResponse struct {
	SummaryID        string  `json:"summary_id"`
	UserID           string  `json:"user_id"`
	SourceType       string  `json:"source_type"`
	SourceID         string  `json:"source_id"`
	LogID            string  `json:"log_id,omitempty"`
	CompletionRate   string  `json:"completion_rate"`
	IntensityMatch   string  `json:"intensity_match"`
	RecoveryAdvice   string  `json:"recovery_advice"`
	AnomalyNotes     string  `json:"anomaly_notes"`
	PerformanceNotes string  `json:"performance_notes"`
	NextSuggestion   string  `json:"next_suggestion"`
	DeletedAt        *string `json:"deleted_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type recommendationGenerateRequest struct {
	UserID string `json:"user_id"`
}

type recommendationFeedbackRequest struct {
	UserID string `json:"user_id"`
	Useful string `json:"useful"`
	Reason string `json:"reason"`
}

func NewHTTPServer(
	addr string,
	internalToken string,
	creator SyncJobCreator,
	reader SyncJobReader,
	retryer SyncJobRetryer,
	asynqClient *asynq.Client,
	profileStore UserProfileStore,
	weatherStore WeatherStore,
	weatherProvider WeatherProvider,
	trainingStore TrainingLogStore,
	asyncJobStore AsyncJobStore,
	baselineStore BaselineStore,
	recService RecommendationService,
) *kratoshttp.Server {
	if addr == "" {
		addr = ":8000"
	}

	srv := kratoshttp.NewServer(
		kratoshttp.Address(addr),
		kratoshttp.Filter(middleware.InternalTokenFilter(internalToken)),
	)

	srv.Handle("/internal/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	srv.Handle("/internal/metrics", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("metrics_placeholder 1\n"))
	}))

	srv.Handle("/internal/v1/sync/jobs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if creator == nil || asynqClient == nil {
			http.Error(w, "sync subsystem unavailable", http.StatusServiceUnavailable)
			return
		}

		var req createSyncJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		payload := task.SyncJobPayload{JobID: uuid.NewString(), UserID: req.UserID, Source: req.Source, RetryCount: 0}
		b, err := task.EncodeSyncJobPayload(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := creator.CreateSyncJob(r.Context(), payload.JobID, payload.UserID, payload.Source); err != nil {
			http.Error(w, "create sync job failed", http.StatusInternalServerError)
			return
		}
		_, err = asynqClient.Enqueue(asynq.NewTask(task.TypeSyncJob, b), asynq.Queue("default"))
		if err != nil {
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job_id": payload.JobID, "status": "queued"})
	}))

	srv.HandlePrefix("/internal/v1/sync/jobs/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reader == nil || retryer == nil || asynqClient == nil {
			http.Error(w, "sync subsystem unavailable", http.StatusServiceUnavailable)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/internal/v1/sync/jobs/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		if strings.HasSuffix(path, "/retry") {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			jobID := strings.TrimSuffix(path, "/retry")
			jobID = strings.TrimSuffix(jobID, "/")
			job, err := retryer.RetrySyncJob(r.Context(), jobID)
			if err != nil {
				http.Error(w, "retry sync job failed", http.StatusInternalServerError)
				return
			}
			payload := task.SyncJobPayload{
				JobID:      job.JobID,
				UserID:     job.UserID,
				Source:     job.Source,
				RetryCount: job.RetryCount,
			}
			b, err := task.EncodeSyncJobPayload(payload)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if _, err := asynqClient.Enqueue(asynq.NewTask(task.TypeSyncJob, b), asynq.Queue("default")); err != nil {
				http.Error(w, "enqueue failed", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]any{
				"job_id":      job.JobID,
				"status":      "queued",
				"retry_count": job.RetryCount,
			})
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jobID := strings.TrimSuffix(path, "/")
		job, err := reader.GetSyncJob(r.Context(), jobID)
		if err != nil {
			http.Error(w, "sync job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
	}))

	srv.Handle("/internal/v1/user/profile", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if profileStore == nil {
			http.Error(w, "user profile subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		switch r.Method {
		case http.MethodPost:
			var req userProfileRequest
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(body, &raw); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if _, ok := raw["fitness_level"]; ok {
				http.Error(w, "fitness_level not allowed", http.StatusBadRequest)
				return
			}
			if _, ok := raw["ability_level"]; ok {
				http.Error(w, "ability_level not allowed", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if err := validateUserProfileRequest(req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			profile := storage.UserProfile{
				UserID:         req.UserID,
				Gender:         req.Gender,
				Age:            req.Age,
				HeightCM:       req.HeightCM,
				WeightKG:       req.WeightKG,
				GoalType:       req.GoalType,
				GoalCycle:      req.GoalCycle,
				GoalFrequency:  req.GoalFrequency,
				GoalPace:       req.GoalPace,
				FitnessLevel:   "unknown",
				RunningYears:   req.RunningYears,
				WeeklySessions: req.WeeklySessions,
				WeeklyDistanceKM: req.WeeklyDistanceKM,
				LongestRunKM:   req.LongestRunKM,
				RecentDiscomfort: req.RecentDiscomfort,
				LocationLat:    *req.LocationLat,
				LocationLng:    *req.LocationLng,
				Country:        req.Country,
				Province:       req.Province,
				City:           req.City,
				LocationSource: req.LocationSource,
			}
			if err := profileStore.UpsertUserProfile(r.Context(), profile); err != nil {
				http.Error(w, "save user profile failed", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, profile)
		case http.MethodGet:
			userID := r.URL.Query().Get("user_id")
			if userID == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			profile, err := profileStore.GetUserProfile(r.Context(), userID)
			if err != nil {
				http.Error(w, "user profile not found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, profile)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	srv.Handle("/internal/v1/weather/snapshot", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if weatherStore == nil || profileStore == nil || weatherProvider == nil {
			http.Error(w, "weather subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req weatherSnapshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.UserID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		snapshotDate, err := parseDate(req.Date)
		if err != nil {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
		profile, err := profileStore.GetUserProfile(r.Context(), req.UserID)
		if err != nil {
			http.Error(w, "user profile not found", http.StatusNotFound)
			return
		}
		location := weather.Location{
			Lat:      profile.LocationLat,
			Lng:      profile.LocationLng,
			Country:  profile.Country,
			Province: profile.Province,
			City:     profile.City,
		}
		input, err := weatherProvider.GetSnapshot(r.Context(), location)
		if err != nil {
			http.Error(w, "weather provider error", http.StatusBadRequest)
			return
		}
		risk := weather.ClassifyRisk(input)
		snapshot := storage.WeatherSnapshot{
			UserID:            req.UserID,
			Date:              snapshotDate,
			TemperatureC:      input.TemperatureC,
			FeelsLikeC:        input.FeelsLikeC,
			Humidity:          input.Humidity,
			WindSpeedMS:       input.WindSpeedMS,
			PrecipitationProb: input.PrecipitationProb,
			AQI:               input.AQI,
			UVIndex:           input.UVIndex,
			RiskLevel:         string(risk),
		}
		if err := weatherStore.CreateWeatherSnapshot(r.Context(), snapshot); err != nil {
			http.Error(w, "save weather snapshot failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    req.UserID,
			"date":       snapshotDate.Format("2006-01-02"),
			"risk_level": snapshot.RiskLevel,
		})
	}))

	srv.Handle("/internal/v1/weather/risk", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if weatherStore == nil {
			http.Error(w, "weather subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		snapshotDate, err := parseDate(r.URL.Query().Get("date"))
		if err != nil {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
		snapshot, err := weatherStore.GetWeatherSnapshot(r.Context(), userID, snapshotDate)
		if err != nil {
			http.Error(w, "weather snapshot not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    userID,
			"date":       snapshotDate.Format("2006-01-02"),
			"risk_level": snapshot.RiskLevel,
		})
	}))

	srv.Handle("/internal/v1/training/logs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if trainingStore == nil {
			http.Error(w, "training subsystem unavailable", http.StatusServiceUnavailable)
			return
		}

		switch r.Method {
		case http.MethodPost:
			var req trainingLogRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			log, start, end, err := buildTrainingLog(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			conflict, err := trainingStore.HasTrainingConflict(r.Context(), req.UserID, start, end, "")
			if err != nil {
				http.Error(w, "conflict check failed", http.StatusInternalServerError)
				return
			}
			if conflict {
				http.Error(w, "training time conflict", http.StatusConflict)
				return
			}
			log.LogID = uuid.NewString()
			log.Source = "manual"
			if err := trainingStore.CreateTrainingLog(r.Context(), log); err != nil {
				http.Error(w, "create training log failed", http.StatusInternalServerError)
				return
			}
			jobID, err := enqueueBaselineRecalc(r.Context(), asyncJobStore, asynqClient, log.UserID, "training_create", log.LogID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"log_id": log.LogID, "job_id": jobID})
		case http.MethodGet:
			userID := r.URL.Query().Get("user_id")
			if userID == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			from, err := parseRangeTime(r.URL.Query().Get("from"), false)
			if err != nil {
				http.Error(w, "invalid from", http.StatusBadRequest)
				return
			}
			to, err := parseRangeTime(r.URL.Query().Get("to"), true)
			if err != nil {
				http.Error(w, "invalid to", http.StatusBadRequest)
				return
			}
			if from.IsZero() {
				from = time.Unix(0, 0)
			}
			if to.IsZero() {
				to = time.Now()
			}
			logs, err := trainingStore.ListTrainingLogs(r.Context(), userID, from, to)
			if err != nil {
				http.Error(w, "list training logs failed", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, logs)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	srv.HandlePrefix("/internal/v1/training/logs/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if trainingStore == nil {
			http.Error(w, "training subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/internal/v1/training/logs/")
		logID := strings.TrimSuffix(path, "/")
		if logID == "" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodPut:
			var req trainingLogRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			log, start, end, err := buildTrainingLog(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			conflict, err := trainingStore.HasTrainingConflict(r.Context(), req.UserID, start, end, logID)
			if err != nil {
				http.Error(w, "conflict check failed", http.StatusInternalServerError)
				return
			}
			if conflict {
				http.Error(w, "training time conflict", http.StatusConflict)
				return
			}
			log.LogID = logID
			log.Source = "manual"
			if err := trainingStore.UpdateTrainingLog(r.Context(), log); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "training log not found", http.StatusNotFound)
					return
				}
				http.Error(w, "update training log failed", http.StatusInternalServerError)
				return
			}
			jobID, err := enqueueBaselineRecalc(r.Context(), asyncJobStore, asynqClient, log.UserID, "training_update", log.LogID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"log_id": log.LogID, "job_id": jobID})
		case http.MethodDelete:
			log, err := trainingStore.GetTrainingLog(r.Context(), logID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "training log not found", http.StatusNotFound)
					return
				}
				http.Error(w, "training log not found", http.StatusNotFound)
				return
			}
			if err := trainingStore.SoftDeleteTrainingLog(r.Context(), logID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "training log not found", http.StatusNotFound)
					return
				}
				http.Error(w, "delete training log failed", http.StatusInternalServerError)
				return
			}
			jobID, err := enqueueBaselineRecalc(r.Context(), asyncJobStore, asynqClient, log.UserID, "training_delete", logID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"log_id": logID, "job_id": jobID})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	srv.Handle("/internal/v1/recommendations/generate", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recService == nil {
			http.Error(w, "recommendation subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req recommendationGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.UserID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		rec, err := recService.Generate(r.Context(), req.UserID)
		if err != nil {
			http.Error(w, "generate recommendation failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, formatRecommendationResponse(rec))
	}))

	srv.Handle("/internal/v1/recommendations/latest", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recService == nil {
			http.Error(w, "recommendation subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		rec, err := recService.GetLatest(r.Context(), userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "recommendation not found", http.StatusNotFound)
				return
			}
			http.Error(w, "get recommendation failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, formatRecommendationResponse(rec))
	}))

	srv.HandlePrefix("/internal/v1/recommendations/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recService == nil {
			http.Error(w, "recommendation subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/internal/v1/recommendations/")
		if !strings.HasSuffix(path, "/feedback") {
			http.NotFound(w, r)
			return
		}
		recID := strings.TrimSuffix(path, "/feedback")
		recID = strings.TrimSuffix(recID, "/")
		if recID == "" {
			http.NotFound(w, r)
			return
		}

		var req recommendationFeedbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.UserID == "" || req.Useful == "" {
			http.Error(w, "user_id/useful required", http.StatusBadRequest)
			return
		}
		if err := recService.Feedback(r.Context(), recID, req.UserID, req.Useful, req.Reason); err != nil {
			if errors.Is(err, recommendation.ErrFeedbackExists) {
				http.Error(w, "feedback exists", http.StatusConflict)
				return
			}
			http.Error(w, "submit feedback failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rec_id": recID})
	}))

	srv.Handle("/internal/v1/baseline/current", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if baselineStore == nil {
			http.Error(w, "baseline subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		current, err := baselineStore.GetBaselineCurrent(r.Context(), userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "baseline not found", http.StatusNotFound)
				return
			}
			http.Error(w, "get baseline failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, current)
	}))

	srv.Handle("/internal/v1/baseline/history", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if baselineStore == nil {
			http.Error(w, "baseline subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		from, err := parseRangeTime(r.URL.Query().Get("from"), false)
		if err != nil {
			http.Error(w, "invalid from", http.StatusBadRequest)
			return
		}
		to, err := parseRangeTime(r.URL.Query().Get("to"), true)
		if err != nil {
			http.Error(w, "invalid to", http.StatusBadRequest)
			return
		}
		if from.IsZero() {
			from = time.Unix(0, 0)
		}
		if to.IsZero() {
			to = time.Now()
		}
		histories, err := baselineStore.ListBaselineHistory(r.Context(), userID, from, to)
		if err != nil {
			http.Error(w, "list baseline history failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, histories)
	}))

	srv.Handle("/internal/v1/training/summaries", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if baselineStore == nil {
			http.Error(w, "training summary subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}
		from, err := parseRangeTime(r.URL.Query().Get("from"), false)
		if err != nil {
			http.Error(w, "invalid from", http.StatusBadRequest)
			return
		}
		to, err := parseRangeTime(r.URL.Query().Get("to"), true)
		if err != nil {
			http.Error(w, "invalid to", http.StatusBadRequest)
			return
		}
		if from.IsZero() {
			from = time.Unix(0, 0)
		}
		if to.IsZero() {
			to = time.Now()
		}
		summaries, err := baselineStore.ListTrainingSummaries(r.Context(), userID, from, to)
		if err != nil {
			http.Error(w, "list training summaries failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, formatTrainingSummaries(summaries))
	}))

	srv.Handle("/internal/v1/training/feedback", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if baselineStore == nil {
			http.Error(w, "training feedback subsystem unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req trainingFeedbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.UserID == "" || req.SourceType == "" || req.SourceID == "" || req.Content == "" {
			http.Error(w, "user_id/source_type/source_id/content required", http.StatusBadRequest)
			return
		}
		if req.SourceType != "log" && req.SourceType != "activity" {
			http.Error(w, "source_type invalid", http.StatusBadRequest)
			return
		}
		logID := ""
		if req.SourceType == "log" {
			logID = req.SourceID
		}
		feedback := storage.TrainingFeedback{
			FeedbackID: uuid.NewString(),
			UserID:     req.UserID,
			SourceType: req.SourceType,
			SourceID:   req.SourceID,
			LogID:      logID,
			Content:    req.Content,
		}
		if err := baselineStore.CreateTrainingFeedback(r.Context(), feedback); err != nil {
			http.Error(w, "create training feedback failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"feedback_id": feedback.FeedbackID})
	}))

	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func formatRecommendationResponse(rec storage.Recommendation) map[string]any {
	return map[string]any{
		"rec_id":              rec.RecID,
		"user_id":             rec.UserID,
		"recommendation_date": rec.RecommendationDate.Format("2006-01-02"),
		"created_at":          rec.CreatedAt,
		"input_json":          rawJSON(rec.InputJSON),
		"output_json":         rawJSON(rec.OutputJSON),
		"override_json":       rawJSON(rec.OverrideJSON),
		"risk_level":          rec.RiskLevel,
		"is_fallback":         rec.IsFallback,
		"ai_provider":         rec.AIProvider,
		"ai_model":            rec.AIModel,
		"prompt_version":      rec.PromptVersion,
		"engine_version":      rec.EngineVersion,
	}
}

func formatTrainingSummaries(items []storage.TrainingSummary) []trainingSummaryResponse {
	out := make([]trainingSummaryResponse, 0, len(items))
	for _, s := range items {
		resp := trainingSummaryResponse{
			SummaryID:        s.SummaryID,
			UserID:           s.UserID,
			SourceType:       s.SourceType,
			SourceID:         s.SourceID,
			LogID:            s.LogID,
			CompletionRate:   s.CompletionRate,
			IntensityMatch:   s.IntensityMatch,
			RecoveryAdvice:   s.RecoveryAdvice,
			AnomalyNotes:     s.AnomalyNotes,
			PerformanceNotes: s.PerformanceNotes,
			NextSuggestion:   s.NextSuggestion,
			CreatedAt:        s.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:        s.UpdatedAt.UTC().Format(time.RFC3339),
		}
		if s.DeletedAt != nil {
			ts := s.DeletedAt.UTC().Format(time.RFC3339)
			resp.DeletedAt = &ts
		}
		out = append(out, resp)
	}
	return out
}

func rawJSON(input []byte) json.RawMessage {
	if len(input) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(input)
}

func buildTrainingLog(req trainingLogRequest) (storage.TrainingLog, time.Time, time.Time, error) {
	if req.UserID == "" {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("user_id required")
	}
	startTime, err := parseStartTime(req.StartTime)
	if err != nil {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("start_time invalid")
	}
	durationSec, err := training.ParseDuration(req.Duration)
	if err != nil || durationSec <= 0 {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("duration invalid")
	}
	if req.DistanceKM <= 0 {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("distance_km invalid")
	}
	paceSec, err := training.ParsePace(req.Pace)
	if err != nil {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("pace invalid")
	}
	if req.RPE < 1 || req.RPE > 10 {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("rpe invalid")
	}

	trainingType, custom, err := resolveTrainingType(req)
	if err != nil {
		return storage.TrainingLog{}, time.Time{}, time.Time{}, errBadRequest("training_type invalid")
	}

	log := storage.TrainingLog{
		UserID:             req.UserID,
		TrainingType:       trainingType,
		TrainingTypeCustom: custom,
		StartTime:          startTime,
		DurationSec:        durationSec,
		DistanceKM:         req.DistanceKM,
		PaceStr:            req.Pace,
		PaceSecPerKM:       paceSec,
		RPE:                req.RPE,
		Discomfort:         req.Discomfort,
	}
	endTime := startTime.Add(time.Duration(durationSec) * time.Second)
	return log, startTime, endTime, nil
}

func resolveTrainingType(req trainingLogRequest) (string, string, error) {
	if req.TrainingTypeCustom != "" {
		return "custom", req.TrainingTypeCustom, nil
	}
	return training.NormalizeTrainingType(req.TrainingType)
}

func parseStartTime(input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, errBadRequest("start_time required")
	}
	return time.ParseInLocation("2006-01-02 15:04:05", input, time.Local)
}

func parseRangeTime(input string, isEnd bool) (time.Time, error) {
	if input == "" {
		return time.Time{}, nil
	}
	if len(input) == 10 {
		t, err := time.ParseInLocation("2006-01-02", input, time.Local)
		if err != nil {
			return time.Time{}, err
		}
		if isEnd {
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		return t, nil
	}
	return time.ParseInLocation("2006-01-02 15:04:05", input, time.Local)
}

func enqueueBaselineRecalc(ctx context.Context, asyncJobStore AsyncJobStore, asynqClient *asynq.Client, userID string, triggerType string, triggerRef string) (string, error) {
	if asyncJobStore == nil {
		return "", errBadRequest("async subsystem unavailable")
	}
	jobID := uuid.NewString()
	payload := task.BaselineRecalcPayload{
		JobID:       jobID,
		UserID:      userID,
		TriggerType: triggerType,
		TriggerRef:  triggerRef,
	}
	b, err := task.EncodeBaselineRecalcPayload(payload)
	if err != nil {
		return "", errBadRequest("payload invalid")
	}
	job := storage.AsyncJob{
		JobID:        jobID,
		JobType:      task.TypeBaselineRecalc,
		UserID:       userID,
		PayloadJSON:  b,
		Status:       "queued",
		RetryCount:   0,
		ErrorMessage: "",
	}
	if err := asyncJobStore.CreateAsyncJob(ctx, job); err != nil {
		return "", errBadRequest("create async job failed")
	}
	if asynqClient == nil {
		_ = asyncJobStore.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, "enqueue client unavailable")
		return "", errBadRequest("enqueue client unavailable")
	}
	if _, err := asynqClient.Enqueue(asynq.NewTask(task.TypeBaselineRecalc, b), asynq.Queue("default")); err != nil {
		_ = asyncJobStore.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, err.Error())
		return "", errBadRequest("enqueue failed")
	}
	return jobID, nil
}

func enqueueTrainingRecalc(ctx context.Context, asyncJobStore AsyncJobStore, asynqClient *asynq.Client, userID string, logID string, op string) (string, error) {
	if asyncJobStore == nil {
		return "", errBadRequest("async subsystem unavailable")
	}
	jobID := uuid.NewString()
	payload := task.TrainingRecalcPayload{
		JobID:     jobID,
		UserID:    userID,
		LogID:     logID,
		Operation: op,
	}
	b, err := task.EncodeTrainingRecalcPayload(payload)
	if err != nil {
		return "", errBadRequest("payload invalid")
	}
	job := storage.AsyncJob{
		JobID:        jobID,
		JobType:      task.TypeTrainingRecalc,
		UserID:       userID,
		PayloadJSON:  b,
		Status:       "queued",
		RetryCount:   0,
		ErrorMessage: "",
	}
	if err := asyncJobStore.CreateAsyncJob(ctx, job); err != nil {
		return "", errBadRequest("create async job failed")
	}
	if asynqClient == nil {
		_ = asyncJobStore.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, "enqueue client unavailable")
		return "", errBadRequest("enqueue client unavailable")
	}
	if _, err := asynqClient.Enqueue(asynq.NewTask(task.TypeTrainingRecalc, b), asynq.Queue("default")); err != nil {
		_ = asyncJobStore.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, err.Error())
		return "", errBadRequest("enqueue failed")
	}
	return jobID, nil
}

func validateUserProfileRequest(req userProfileRequest) error {
	if req.UserID == "" {
		return errBadRequest("user_id required")
	}
	if req.LocationLat == nil || req.LocationLng == nil {
		return errBadRequest("location required")
	}
	if *req.LocationLat < -90 || *req.LocationLat > 90 {
		return errBadRequest("location_lat invalid")
	}
	if *req.LocationLng < -180 || *req.LocationLng > 180 {
		return errBadRequest("location_lng invalid")
	}
	if req.Country == "" || req.Province == "" || req.City == "" {
		return errBadRequest("country/province/city required")
	}
	if req.LocationSource != "geo" && req.LocationSource != "manual" {
		return errBadRequest("location_source invalid")
	}
	if req.Gender == "" || req.Age <= 0 || req.HeightCM <= 0 || req.WeightKG <= 0 {
		return errBadRequest("basic profile required")
	}
	if req.GoalType == "" || req.GoalCycle == "" || req.GoalFrequency <= 0 || req.GoalPace == "" {
		return errBadRequest("goal required")
	}
	if req.RunningYears == "" || req.WeeklySessions == "" || req.WeeklyDistanceKM == "" || req.LongestRunKM == "" || req.RecentDiscomfort == "" {
		return errBadRequest("questionnaire required")
	}
	if !containsString(req.RunningYears, "0", "<1", "1-3", "3+") {
		return errBadRequest("running_years invalid")
	}
	if !containsString(req.WeeklySessions, "0-1", "2-3", "4+") {
		return errBadRequest("weekly_sessions invalid")
	}
	if !containsString(req.WeeklyDistanceKM, "0-5", "5-15", "15-30", "30+") {
		return errBadRequest("weekly_distance_km invalid")
	}
	if !containsString(req.LongestRunKM, "0", "3", "5", "10", "21") {
		return errBadRequest("longest_run_km invalid")
	}
	if !containsString(req.RecentDiscomfort, "yes", "no") {
		return errBadRequest("recent_discomfort invalid")
	}
	return nil
}

func containsString(input string, choices ...string) bool {
	for _, choice := range choices {
		if input == choice {
			return true
		}
	}
	return false
}

func parseDate(input string) (time.Time, error) {
	if input == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	parsed, err := time.Parse("2006-01-02", input)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

type errBadRequest string

func (e errBadRequest) Error() string {
	return string(e)
}
