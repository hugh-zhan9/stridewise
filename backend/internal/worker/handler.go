package worker

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/task"
)

func HandleSyncJob(_ context.Context, t *asynq.Task) error {
	p, err := task.DecodeSyncJobPayload(t.Payload())
	if err != nil {
		return err
	}
	fmt.Printf("processing sync job: user=%s source=%s\n", p.UserID, p.Source)
	return nil
}
