package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type RawActivity struct {
	SourceActivityID string         `json:"source_activity_id"`
	Name             string         `json:"name"`
	DistanceM        float64        `json:"distance_m"`
	MovingTimeSec    int            `json:"moving_time_sec"`
	StartTime        time.Time      `json:"start_time"`
	SummaryPolyline  string         `json:"summary_polyline"`
	Raw              map[string]any `json:"raw,omitempty"`
}

type Checkpoint struct {
	Cursor       string
	LastSyncedAt time.Time
}

type FetchResult struct {
	Activities   []RawActivity
	NextCursor   string
	LastSyncedAt time.Time
}

type CanonicalActivity struct {
	UserID           string
	Source           string
	SourceActivityID string
	Name             string
	DistanceM        float64
	MovingTimeSec    int
	StartTimeUTC     time.Time
	StartTimeLocal   time.Time
	Timezone         string
	SummaryPolyline  string
	RawJSON          []byte
}

type Connector interface {
	FetchActivities(ctx context.Context, userID string, checkpoint Checkpoint) (FetchResult, error)
}

type Store interface {
	MarkRunning(ctx context.Context, jobID string) error
	SaveRawAndCanonical(ctx context.Context, jobID string, userID string, source string, activities []CanonicalActivity) error
	MarkSuccess(ctx context.Context, jobID string, fetchedCount int) error
	MarkFailed(ctx context.Context, jobID string, retryCount int, errorMessage string) error
	GetCheckpoint(ctx context.Context, userID, source string) (Checkpoint, error)
	UpsertCheckpoint(ctx context.Context, userID, source string, cp Checkpoint) error
	AppendSyncError(ctx context.Context, jobID, source, errorMessage string, retryable bool) error
}

type BaselineRecalcEnqueuer interface {
	EnqueueBaselineRecalc(ctx context.Context, userID, triggerType, triggerRef string) error
}

type Processor struct {
	store            Store
	connectors       map[string]Connector
	baselineEnqueuer BaselineRecalcEnqueuer
}

func NewProcessor(store Store, connectors map[string]Connector) *Processor {
	return &Processor{store: store, connectors: connectors}
}

func (p *Processor) SetBaselineEnqueuer(enqueuer BaselineRecalcEnqueuer) {
	p.baselineEnqueuer = enqueuer
}

func (p *Processor) ProcessSyncJob(ctx context.Context, jobID, userID, source string, retryCount int) error {
	if err := p.store.MarkRunning(ctx, jobID); err != nil {
		return err
	}

	connector, ok := p.connectors[source]
	if !ok {
		err := fmt.Errorf("connector not found: %s", source)
		_ = p.store.MarkFailed(ctx, jobID, retryCount, err.Error())
		return err
	}

	checkpoint, err := p.store.GetCheckpoint(ctx, userID, source)
	if err != nil {
		_ = p.store.MarkFailed(ctx, jobID, retryCount, err.Error())
		return err
	}

	result, err := connector.FetchActivities(ctx, userID, checkpoint)
	if err != nil {
		_ = p.store.AppendSyncError(ctx, jobID, source, err.Error(), retryCount < 5)
		_ = p.store.MarkFailed(ctx, jobID, retryCount, err.Error())
		return err
	}

	canonical := make([]CanonicalActivity, 0, len(result.Activities))
	for _, a := range result.Activities {
		rawJSON, _ := json.Marshal(a.Raw)
		canonical = append(canonical, CanonicalActivity{
			UserID:           userID,
			Source:           source,
			SourceActivityID: a.SourceActivityID,
			Name:             a.Name,
			DistanceM:        a.DistanceM,
			MovingTimeSec:    a.MovingTimeSec,
			StartTimeUTC:     a.StartTime.UTC(),
			StartTimeLocal:   a.StartTime,
			Timezone:         "UTC",
			SummaryPolyline:  a.SummaryPolyline,
			RawJSON:          rawJSON,
		})
	}

	if err := p.store.SaveRawAndCanonical(ctx, jobID, userID, source, canonical); err != nil {
		_ = p.store.MarkFailed(ctx, jobID, retryCount, err.Error())
		return err
	}
	if err := p.store.UpsertCheckpoint(ctx, userID, source, Checkpoint{
		Cursor:       result.NextCursor,
		LastSyncedAt: result.LastSyncedAt,
	}); err != nil {
		_ = p.store.MarkFailed(ctx, jobID, retryCount, err.Error())
		return err
	}

	if err := p.store.MarkSuccess(ctx, jobID, len(canonical)); err != nil {
		return err
	}
	if p.baselineEnqueuer != nil {
		if err := p.baselineEnqueuer.EnqueueBaselineRecalc(ctx, userID, "sync", jobID); err != nil {
			_ = p.store.AppendSyncError(ctx, jobID, source, "enqueue baseline recalc failed: "+err.Error(), true)
		}
	}
	return nil
}
