package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"stridewise/backend/internal/middleware"
	"stridewise/backend/internal/task"
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

type createSyncJobRequest struct {
	UserID string `json:"user_id"`
	Source string `json:"source"`
}

func NewHTTPServer(
	addr string,
	internalToken string,
	creator SyncJobCreator,
	reader SyncJobReader,
	retryer SyncJobRetryer,
	asynqClient *asynq.Client,
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

	return srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
