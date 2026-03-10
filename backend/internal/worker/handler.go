package worker

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/sync"
	"stridewise/backend/internal/task"
)

var processor *sync.Processor

func SetProcessor(p *sync.Processor) {
	processor = p
}

func HandleSyncJob(ctx context.Context, t *asynq.Task) error {
	if processor == nil {
		return errors.New("sync processor is not configured")
	}

	p, err := task.DecodeSyncJobPayload(t.Payload())
	if err != nil {
		return err
	}

	return processor.ProcessSyncJob(ctx, p.JobID, p.UserID, p.Source, p.RetryCount)
}
