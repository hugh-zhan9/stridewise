package asyncjob

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
)

type AbilityLevelStore interface {
	CreateAsyncJob(ctx context.Context, job storage.AsyncJob) error
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
	FindActiveAsyncJob(ctx context.Context, userID, jobType string) (storage.AsyncJob, error)
}

type AbilityLevelEnqueuer struct {
	store  AbilityLevelStore
	client *asynq.Client
}

func NewAbilityLevelEnqueuer(store AbilityLevelStore, client *asynq.Client) *AbilityLevelEnqueuer {
	return &AbilityLevelEnqueuer{store: store, client: client}
}

func (e *AbilityLevelEnqueuer) EnqueueAbilityLevelCalc(ctx context.Context, userID, triggerType, triggerRef string) (string, error) {
	if e.store == nil {
		return "", errors.New("async store unavailable")
	}
	if e.client == nil {
		return "", errors.New("asynq client unavailable")
	}
	if job, err := e.store.FindActiveAsyncJob(ctx, userID, task.TypeAbilityLevelCalc); err == nil && job.JobID != "" {
		return job.JobID, nil
	}
	jobID := uuid.NewString()
	payload := task.AbilityLevelPayload{
		JobID:       jobID,
		UserID:      userID,
		TriggerType: triggerType,
		TriggerRef:  triggerRef,
	}
	b, err := task.EncodeAbilityLevelPayload(payload)
	if err != nil {
		return "", err
	}
	job := storage.AsyncJob{
		JobID:        jobID,
		JobType:      task.TypeAbilityLevelCalc,
		UserID:       userID,
		PayloadJSON:  b,
		Status:       "queued",
		RetryCount:   0,
		ErrorMessage: "",
	}
	if err := e.store.CreateAsyncJob(ctx, job); err != nil {
		return "", err
	}
	if _, err := e.client.Enqueue(asynq.NewTask(task.TypeAbilityLevelCalc, b), asynq.Queue("default")); err != nil {
		_ = e.store.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, err.Error())
		return "", err
	}
	return jobID, nil
}
