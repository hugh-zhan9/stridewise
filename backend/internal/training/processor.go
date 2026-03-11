package training

import (
	"context"
	"errors"

	"stridewise/backend/internal/storage"
)

type AsyncJobStore interface {
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
}

type BaselineRecalculator interface {
	RecalcForTrigger(ctx context.Context, userID, triggerType, triggerRef string) (error, error)
}

type RecommendationService interface {
	Generate(ctx context.Context, userID string) (storage.Recommendation, error)
}

type Processor struct {
	store       AsyncJobStore
	baseline    BaselineRecalculator
	recommender RecommendationService
}

func NewProcessor(store AsyncJobStore, baseline BaselineRecalculator, recommender RecommendationService) *Processor {
	return &Processor{store: store, baseline: baseline, recommender: recommender}
}

func (p *Processor) ProcessTrainingRecalc(ctx context.Context, jobID, userID, logID, operation string, retryCount int) error {
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "running", retryCount, ""); err != nil {
		return err
	}
	summaryErr, err := p.recalcBaseline(ctx, userID, logID, operation)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	if err := p.refreshRecommendation(ctx, userID, logID, operation); err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	if err := p.rollbackSummaryAndFeedback(ctx, userID, logID, operation); err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	errMsg := ""
	if summaryErr != nil {
		errMsg = summaryErr.Error()
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "success", retryCount, errMsg); err != nil {
		return err
	}
	return nil
}

func (p *Processor) recalcBaseline(ctx context.Context, userID string, logID string, operation string) (error, error) {
	if p.baseline == nil {
		return nil, errors.New("baseline processor not configured")
	}
	triggerType := "training_" + operation
	return p.baseline.RecalcForTrigger(ctx, userID, triggerType, logID)
}

func (p *Processor) refreshRecommendation(ctx context.Context, userID string, _ string, _ string) error {
	if p.recommender == nil {
		return errors.New("recommendation service not configured")
	}
	_, err := p.recommender.Generate(ctx, userID)
	return err
}

func (p *Processor) rollbackSummaryAndFeedback(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
