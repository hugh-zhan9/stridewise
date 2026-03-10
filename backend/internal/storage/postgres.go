package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	syncjob "stridewise/backend/internal/sync"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type SyncJob struct {
	JobID        string
	UserID       string
	Source       string
	Status       string
	RetryCount   int
	ErrorMessage string
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) CreateSyncJob(ctx context.Context, jobID, userID, source string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sync_jobs (job_id, user_id, source, status, retry_count, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, 'queued', 0, '', NOW(), NOW())
	`, jobID, userID, source)
	return err
}

func (s *PostgresStore) MarkRunning(ctx context.Context, jobID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status='running', started_at=COALESCE(started_at, NOW()), updated_at=NOW()
		WHERE job_id=$1
	`, jobID)
	return err
}

func (s *PostgresStore) SaveRawAndCanonical(ctx context.Context, jobID string, userID string, source string, activities []syncjob.CanonicalActivity) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, a := range activities {
		_, err = tx.Exec(ctx, `
			INSERT INTO raw_activities (job_id, user_id, source, source_activity_id, payload_json, fetched_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6)
			ON CONFLICT (user_id, source, source_activity_id)
			DO UPDATE SET payload_json=EXCLUDED.payload_json, fetched_at=EXCLUDED.fetched_at, job_id=EXCLUDED.job_id
		`, jobID, userID, source, a.SourceActivityID, string(a.RawJSON), time.Now().UTC())
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO activities (
				user_id, source, source_activity_id, name,
				distance_m, moving_time_sec, start_time_utc, start_time_local,
				timezone, summary_polyline, created_at, updated_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
			ON CONFLICT (user_id, source, source_activity_id)
			DO UPDATE SET
				name=EXCLUDED.name,
				distance_m=EXCLUDED.distance_m,
				moving_time_sec=EXCLUDED.moving_time_sec,
				start_time_utc=EXCLUDED.start_time_utc,
				start_time_local=EXCLUDED.start_time_local,
				timezone=EXCLUDED.timezone,
				summary_polyline=EXCLUDED.summary_polyline,
				updated_at=NOW()
		`, userID, source, a.SourceActivityID, a.Name, a.DistanceM, a.MovingTimeSec, a.StartTimeUTC, a.StartTimeLocal, a.Timezone, a.SummaryPolyline)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) MarkSuccess(ctx context.Context, jobID string, fetchedCount int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status='success', fetched_count=$2, upserted_count=$2, failed_count=0, finished_at=NOW(), updated_at=NOW(), error_message=''
		WHERE job_id=$1
	`, jobID, fetchedCount)
	return err
}

func (s *PostgresStore) MarkFailed(ctx context.Context, jobID string, retryCount int, errorMessage string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status='failed', retry_count=$2, error_message=$3, finished_at=NOW(), updated_at=NOW()
		WHERE job_id=$1
	`, jobID, retryCount, errorMessage)
	return err
}

func (s *PostgresStore) GetCheckpoint(ctx context.Context, userID, source string) (syncjob.Checkpoint, error) {
	var cp syncjob.Checkpoint
	err := s.pool.QueryRow(ctx, `
		SELECT cursor, last_synced_at_utc
		FROM sync_checkpoints
		WHERE user_id=$1 AND source=$2
	`, userID, source).Scan(&cp.Cursor, &cp.LastSyncedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return syncjob.Checkpoint{}, nil
		}
		return syncjob.Checkpoint{}, err
	}
	return cp, nil
}

func (s *PostgresStore) UpsertCheckpoint(ctx context.Context, userID, source string, cp syncjob.Checkpoint) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sync_checkpoints (user_id, source, cursor, last_synced_at_utc, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, source)
		DO UPDATE SET
			cursor=EXCLUDED.cursor,
			last_synced_at_utc=EXCLUDED.last_synced_at_utc,
			updated_at=NOW()
	`, userID, source, cp.Cursor, cp.LastSyncedAt)
	return err
}

func (s *PostgresStore) AppendSyncError(ctx context.Context, jobID, source, errorMessage string, retryable bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sync_errors (job_id, source, source_activity_id, error_code, error_message, retryable, created_at)
		VALUES ($1, $2, '', '', $3, $4, NOW())
	`, jobID, source, errorMessage, retryable)
	return err
}

func (s *PostgresStore) GetSyncJob(ctx context.Context, jobID string) (SyncJob, error) {
	var job SyncJob
	err := s.pool.QueryRow(ctx, `
		SELECT job_id, user_id, source, status, retry_count, error_message
		FROM sync_jobs
		WHERE job_id=$1
	`, jobID).Scan(&job.JobID, &job.UserID, &job.Source, &job.Status, &job.RetryCount, &job.ErrorMessage)
	if err != nil {
		return SyncJob{}, err
	}
	return job, nil
}

func (s *PostgresStore) RetrySyncJob(ctx context.Context, jobID string) (SyncJob, error) {
	var job SyncJob
	err := s.pool.QueryRow(ctx, `
		UPDATE sync_jobs
		SET status='queued', retry_count=retry_count+1, error_message='', updated_at=NOW()
		WHERE job_id=$1
		RETURNING job_id, user_id, source, status, retry_count, error_message
	`, jobID).Scan(&job.JobID, &job.UserID, &job.Source, &job.Status, &job.RetryCount, &job.ErrorMessage)
	if err != nil {
		return SyncJob{}, err
	}
	return job, nil
}
