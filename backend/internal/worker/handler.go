package worker

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/baseline"
	"stridewise/backend/internal/ability"
	"stridewise/backend/internal/sync"
	"stridewise/backend/internal/task"
	"stridewise/backend/internal/training"
)

var syncProcessor *sync.Processor
var trainingProcessor *training.Processor
var baselineProcessor *baseline.Processor
var abilityProcessor *ability.Processor

func SetSyncProcessor(p *sync.Processor) {
	syncProcessor = p
}

func SetTrainingProcessor(p *training.Processor) {
	trainingProcessor = p
}

func SetBaselineProcessor(p *baseline.Processor) {
	baselineProcessor = p
}

func SetAbilityProcessor(p *ability.Processor) {
	abilityProcessor = p
}

func HandleSyncJob(ctx context.Context, t *asynq.Task) error {
	if syncProcessor == nil {
		return errors.New("sync processor is not configured")
	}

	p, err := task.DecodeSyncJobPayload(t.Payload())
	if err != nil {
		return err
	}

	return syncProcessor.ProcessSyncJob(ctx, p.JobID, p.UserID, p.Source, p.RetryCount)
}

func HandleTrainingRecalc(ctx context.Context, t *asynq.Task) error {
	if trainingProcessor == nil {
		return errors.New("training processor is not configured")
	}
	p, err := task.DecodeTrainingRecalcPayload(t.Payload())
	if err != nil {
		return err
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	return trainingProcessor.ProcessTrainingRecalc(ctx, p.JobID, p.UserID, p.LogID, p.Operation, retryCount)
}

func HandleBaselineRecalc(ctx context.Context, t *asynq.Task) error {
	if baselineProcessor == nil {
		return errors.New("baseline processor is not configured")
	}
	p, err := task.DecodeBaselineRecalcPayload(t.Payload())
	if err != nil {
		return err
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	return baselineProcessor.ProcessBaselineRecalc(ctx, p.JobID, p.UserID, p.TriggerType, p.TriggerRef, retryCount)
}

func HandleAbilityLevelCalc(ctx context.Context, t *asynq.Task) error {
	if abilityProcessor == nil {
		return errors.New("ability processor is not configured")
	}
	p, err := task.DecodeAbilityLevelPayload(t.Payload())
	if err != nil {
		return err
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	return abilityProcessor.ProcessAbilityLevel(ctx, p.JobID, p.UserID, p.TriggerType, p.TriggerRef, retryCount)
}
