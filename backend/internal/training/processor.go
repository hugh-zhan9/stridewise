package training

import "context"

type AsyncJobStore interface {
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
}

type Processor struct {
	store AsyncJobStore
}

func NewProcessor(store AsyncJobStore) *Processor {
	return &Processor{store: store}
}

func (p *Processor) ProcessTrainingRecalc(ctx context.Context, jobID, userID, logID, operation string, retryCount int) error {
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "running", retryCount, ""); err != nil {
		return err
	}
	if err := p.recalcBaseline(ctx, userID, logID, operation); err != nil {
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
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "success", retryCount, ""); err != nil {
		return err
	}
	return nil
}

func (p *Processor) recalcBaseline(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (p *Processor) refreshRecommendation(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (p *Processor) rollbackSummaryAndFeedback(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
