package asyncjob

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
)

type Store interface {
	CreateAsyncJob(ctx context.Context, job storage.AsyncJob) error
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
}

type BaselineEnqueuer struct {
	store  Store
	client *asynq.Client
}

func NewBaselineEnqueuer(store Store, client *asynq.Client) *BaselineEnqueuer {
	return &BaselineEnqueuer{store: store, client: client}
}

func (e *BaselineEnqueuer) EnqueueBaselineRecalc(ctx context.Context, userID, triggerType, triggerRef string) error {
	if e.store == nil {
		return errors.New("async store unavailable")
	}
	if e.client == nil {
		return errors.New("asynq client unavailable")
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
		return err
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
	if err := e.store.CreateAsyncJob(ctx, job); err != nil {
		return err
	}
	if _, err := e.client.Enqueue(asynq.NewTask(task.TypeBaselineRecalc, b), asynq.Queue("default")); err != nil {
		_ = e.store.UpdateAsyncJobStatus(ctx, jobID, "failed", 0, err.Error())
		return err
	}
	return nil
}
