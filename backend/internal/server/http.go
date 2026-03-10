package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"stridewise/backend/internal/middleware"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
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
	FitnessLevel   string   `json:"fitness_level"`
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
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
				FitnessLevel:   req.FitnessLevel,
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

	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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
	if req.FitnessLevel == "" {
		return errBadRequest("fitness_level required")
	}
	return nil
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
