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

type UserProfile struct {
	UserID         string
	Gender         string
	Age            int
	HeightCM       int
	WeightKG       int
	GoalType       string
	GoalCycle      string
	GoalFrequency  int
	GoalPace       string
	FitnessLevel   string
	LocationLat    float64
	LocationLng    float64
	Country        string
	Province       string
	City           string
	LocationSource string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WeatherSnapshot struct {
	UserID            string
	Date              time.Time
	TemperatureC      float64
	FeelsLikeC        float64
	Humidity          float64
	WindSpeedMS       float64
	PrecipitationProb float64
	AQI               int
	UVIndex           float64
	RiskLevel         string
	CreatedAt         time.Time
}

type TrainingLog struct {
	LogID              string
	UserID             string
	Source             string
	TrainingType       string
	TrainingTypeCustom string
	StartTime          time.Time
	DurationSec        int
	DistanceKM         float64
	PaceStr            string
	PaceSecPerKM       int
	RPE                int
	Discomfort         bool
	DeletedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Recommendation struct {
	RecID              string
	UserID             string
	RecommendationDate time.Time
	InputJSON          []byte
	OutputJSON         []byte
	RiskLevel          string
	OverrideJSON       []byte
	IsFallback         bool
	AIProvider         string
	AIModel            string
	PromptVersion      string
	EngineVersion      string
	CreatedAt          time.Time
}

type RecommendationFeedback struct {
	FeedbackID string
	RecID      string
	UserID     string
	Useful     string
	Reason     string
	CreatedAt  time.Time
}

type Activity struct {
	UserID         string
	DistanceM      float64
	MovingTimeSec  int
	StartTimeLocal time.Time
}

type BaselineCurrent struct {
	UserID            string
	ComputedAt        time.Time
	DataSessions7d    int
	AcuteLoadSRPE     float64
	ChronicLoadSRPE   float64
	ACWRSRPE          float64
	AcuteLoadDistance float64
	ChronicLoadDistance float64
	ACWRDistance      float64
	Monotony          float64
	Strain            float64
	PaceAvgSecPerKM   int
	PaceLowSecPerKM   int
	PaceHighSecPerKM  int
	Status            string
}

type BaselineHistory struct {
	BaselineID        string
	UserID            string
	ComputedAt        time.Time
	TriggerType       string
	TriggerRef        string
	DataSessions7d    int
	AcuteLoadSRPE     float64
	ChronicLoadSRPE   float64
	ACWRSRPE          float64
	AcuteLoadDistance float64
	ChronicLoadDistance float64
	ACWRDistance      float64
	Monotony          float64
	Strain            float64
	PaceAvgSecPerKM   int
	PaceLowSecPerKM   int
	PaceHighSecPerKM  int
	Status            string
}

type TrainingSummary struct {
	SummaryID        string
	UserID           string
	LogID            string
	CompletionRate   string
	IntensityMatch   string
	RecoveryAdvice   string
	AnomalyNotes     string
	PerformanceNotes string
	NextSuggestion   string
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TrainingFeedback struct {
	FeedbackID string
	UserID     string
	LogID      string
	Content    string
	CreatedAt  time.Time
}

type AsyncJob struct {
	JobID        string
	JobType      string
	UserID       string
	PayloadJSON  []byte
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

func (s *PostgresStore) CreateTrainingLog(ctx context.Context, log TrainingLog) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO training_logs (
			log_id, user_id, source, training_type, training_type_custom,
			start_time, duration_sec, distance_km, pace_str, pace_sec_per_km,
			rpe, discomfort, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
	`, log.LogID, log.UserID, log.Source, log.TrainingType, log.TrainingTypeCustom,
		log.StartTime, log.DurationSec, log.DistanceKM, log.PaceStr, log.PaceSecPerKM,
		log.RPE, log.Discomfort)
	return err
}

func (s *PostgresStore) UpdateTrainingLog(ctx context.Context, log TrainingLog) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE training_logs
		SET training_type=$2, training_type_custom=$3, start_time=$4, duration_sec=$5,
		    distance_km=$6, pace_str=$7, pace_sec_per_km=$8, rpe=$9, discomfort=$10,
		    updated_at=NOW()
		WHERE log_id=$1 AND deleted_at IS NULL
	`, log.LogID, log.TrainingType, log.TrainingTypeCustom, log.StartTime, log.DurationSec,
		log.DistanceKM, log.PaceStr, log.PaceSecPerKM, log.RPE, log.Discomfort)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *PostgresStore) SoftDeleteTrainingLog(ctx context.Context, logID string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE training_logs
		SET deleted_at=NOW(), updated_at=NOW()
		WHERE log_id=$1 AND deleted_at IS NULL
	`, logID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *PostgresStore) CreateAsyncJob(ctx context.Context, job AsyncJob) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO async_jobs (
			job_id, job_type, user_id, payload_json, status, retry_count, error_message, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,NOW(),NOW())
	`, job.JobID, job.JobType, job.UserID, string(job.PayloadJSON), job.Status, job.RetryCount, job.ErrorMessage)
	return err
}

func (s *PostgresStore) UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE async_jobs
		SET status=$2, retry_count=$3, error_message=$4, updated_at=NOW()
		WHERE job_id=$1
	`, jobID, status, retryCount, errMsg)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *PostgresStore) UpsertBaselineCurrent(ctx context.Context, b BaselineCurrent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO baseline_current (
			user_id, computed_at, data_sessions_7d, acute_load_srpe, chronic_load_srpe, acwr_srpe,
			acute_load_distance, chronic_load_distance, acwr_distance, monotony, strain,
			pace_avg_sec_per_km, pace_low_sec_per_km, pace_high_sec_per_km, status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (user_id)
		DO UPDATE SET
			computed_at=EXCLUDED.computed_at,
			data_sessions_7d=EXCLUDED.data_sessions_7d,
			acute_load_srpe=EXCLUDED.acute_load_srpe,
			chronic_load_srpe=EXCLUDED.chronic_load_srpe,
			acwr_srpe=EXCLUDED.acwr_srpe,
			acute_load_distance=EXCLUDED.acute_load_distance,
			chronic_load_distance=EXCLUDED.chronic_load_distance,
			acwr_distance=EXCLUDED.acwr_distance,
			monotony=EXCLUDED.monotony,
			strain=EXCLUDED.strain,
			pace_avg_sec_per_km=EXCLUDED.pace_avg_sec_per_km,
			pace_low_sec_per_km=EXCLUDED.pace_low_sec_per_km,
			pace_high_sec_per_km=EXCLUDED.pace_high_sec_per_km,
			status=EXCLUDED.status
	`, b.UserID, b.ComputedAt, b.DataSessions7d, b.AcuteLoadSRPE, b.ChronicLoadSRPE, b.ACWRSRPE,
		b.AcuteLoadDistance, b.ChronicLoadDistance, b.ACWRDistance, b.Monotony, b.Strain,
		b.PaceAvgSecPerKM, b.PaceLowSecPerKM, b.PaceHighSecPerKM, b.Status)
	return err
}

func (s *PostgresStore) CreateBaselineHistory(ctx context.Context, b BaselineHistory) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO baseline_history (
			baseline_id, user_id, computed_at, trigger_type, trigger_ref, data_sessions_7d,
			acute_load_srpe, chronic_load_srpe, acwr_srpe, acute_load_distance,
			chronic_load_distance, acwr_distance, monotony, strain,
			pace_avg_sec_per_km, pace_low_sec_per_km, pace_high_sec_per_km, status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, b.BaselineID, b.UserID, b.ComputedAt, b.TriggerType, b.TriggerRef, b.DataSessions7d,
		b.AcuteLoadSRPE, b.ChronicLoadSRPE, b.ACWRSRPE, b.AcuteLoadDistance,
		b.ChronicLoadDistance, b.ACWRDistance, b.Monotony, b.Strain,
		b.PaceAvgSecPerKM, b.PaceLowSecPerKM, b.PaceHighSecPerKM, b.Status)
	return err
}

func (s *PostgresStore) ListBaselineHistory(ctx context.Context, userID string, from time.Time, to time.Time) ([]BaselineHistory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT baseline_id, user_id, computed_at, trigger_type, trigger_ref, data_sessions_7d,
		       acute_load_srpe, chronic_load_srpe, acwr_srpe, acute_load_distance,
		       chronic_load_distance, acwr_distance, monotony, strain,
		       pace_avg_sec_per_km, pace_low_sec_per_km, pace_high_sec_per_km, status
		FROM baseline_history
		WHERE user_id=$1 AND computed_at >= $2 AND computed_at <= $3
		ORDER BY computed_at DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BaselineHistory
	for rows.Next() {
		var b BaselineHistory
		if err := rows.Scan(
			&b.BaselineID, &b.UserID, &b.ComputedAt, &b.TriggerType, &b.TriggerRef, &b.DataSessions7d,
			&b.AcuteLoadSRPE, &b.ChronicLoadSRPE, &b.ACWRSRPE, &b.AcuteLoadDistance,
			&b.ChronicLoadDistance, &b.ACWRDistance, &b.Monotony, &b.Strain,
			&b.PaceAvgSecPerKM, &b.PaceLowSecPerKM, &b.PaceHighSecPerKM, &b.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (s *PostgresStore) GetBaselineCurrent(ctx context.Context, userID string) (BaselineCurrent, error) {
	var b BaselineCurrent
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, computed_at, data_sessions_7d, acute_load_srpe, chronic_load_srpe, acwr_srpe,
		       acute_load_distance, chronic_load_distance, acwr_distance, monotony, strain,
		       pace_avg_sec_per_km, pace_low_sec_per_km, pace_high_sec_per_km, status
		FROM baseline_current
		WHERE user_id=$1
	`, userID).Scan(
		&b.UserID, &b.ComputedAt, &b.DataSessions7d, &b.AcuteLoadSRPE, &b.ChronicLoadSRPE, &b.ACWRSRPE,
		&b.AcuteLoadDistance, &b.ChronicLoadDistance, &b.ACWRDistance, &b.Monotony, &b.Strain,
		&b.PaceAvgSecPerKM, &b.PaceLowSecPerKM, &b.PaceHighSecPerKM, &b.Status,
	)
	if err != nil {
		return BaselineCurrent{}, err
	}
	return b, nil
}

func (s *PostgresStore) UpsertTrainingSummary(ctx context.Context, summary TrainingSummary) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO training_summaries (
			summary_id, user_id, log_id, completion_rate, intensity_match, recovery_advice,
			anomaly_notes, performance_notes, next_suggestion, deleted_at, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
		ON CONFLICT (log_id)
		DO UPDATE SET
			completion_rate=EXCLUDED.completion_rate,
			intensity_match=EXCLUDED.intensity_match,
			recovery_advice=EXCLUDED.recovery_advice,
			anomaly_notes=EXCLUDED.anomaly_notes,
			performance_notes=EXCLUDED.performance_notes,
			next_suggestion=EXCLUDED.next_suggestion,
			deleted_at=EXCLUDED.deleted_at,
			updated_at=NOW()
	`, summary.SummaryID, summary.UserID, summary.LogID, summary.CompletionRate, summary.IntensityMatch, summary.RecoveryAdvice,
		summary.AnomalyNotes, summary.PerformanceNotes, summary.NextSuggestion, summary.DeletedAt)
	return err
}

func (s *PostgresStore) GetTrainingSummary(ctx context.Context, logID string) (TrainingSummary, error) {
	var out TrainingSummary
	err := s.pool.QueryRow(ctx, `
		SELECT summary_id, user_id, log_id, completion_rate, intensity_match,
		       recovery_advice, anomaly_notes, performance_notes, next_suggestion,
		       deleted_at, created_at, updated_at
		FROM training_summaries
		WHERE log_id=$1
	`, logID).Scan(
		&out.SummaryID, &out.UserID, &out.LogID, &out.CompletionRate, &out.IntensityMatch,
		&out.RecoveryAdvice, &out.AnomalyNotes, &out.PerformanceNotes, &out.NextSuggestion,
		&out.DeletedAt, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return TrainingSummary{}, err
	}
	return out, nil
}

func (s *PostgresStore) ListTrainingSummaries(ctx context.Context, userID string, from time.Time, to time.Time) ([]TrainingSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.summary_id, s.user_id, s.log_id, s.completion_rate, s.intensity_match,
		       s.recovery_advice, s.anomaly_notes, s.performance_notes, s.next_suggestion,
		       s.deleted_at, s.created_at, s.updated_at
		FROM training_summaries s
		JOIN training_logs l ON l.log_id = s.log_id
		WHERE s.user_id=$1 AND s.deleted_at IS NULL AND l.deleted_at IS NULL AND l.start_time >= $2 AND l.start_time <= $3
		ORDER BY l.start_time DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrainingSummary
	for rows.Next() {
		var s2 TrainingSummary
		if err := rows.Scan(
			&s2.SummaryID, &s2.UserID, &s2.LogID, &s2.CompletionRate, &s2.IntensityMatch,
			&s2.RecoveryAdvice, &s2.AnomalyNotes, &s2.PerformanceNotes, &s2.NextSuggestion,
			&s2.DeletedAt, &s2.CreatedAt, &s2.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s2)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (s *PostgresStore) CreateTrainingFeedback(ctx context.Context, feedback TrainingFeedback) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO training_feedbacks (feedback_id, user_id, log_id, content, created_at)
		VALUES ($1,$2,$3,$4,NOW())
	`, feedback.FeedbackID, feedback.UserID, feedback.LogID, feedback.Content)
	return err
}

func (s *PostgresStore) CreateRecommendation(ctx context.Context, rec Recommendation) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO recommendations (
			rec_id, user_id, recommendation_date, input_json, output_json, risk_level,
			override_json, is_fallback, ai_provider, ai_model, prompt_version, engine_version, created_at
		)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7::jsonb,$8,$9,$10,$11,$12,NOW())
	`, rec.RecID, rec.UserID, rec.RecommendationDate, string(rec.InputJSON), string(rec.OutputJSON),
		rec.RiskLevel, string(rec.OverrideJSON), rec.IsFallback, rec.AIProvider, rec.AIModel, rec.PromptVersion, rec.EngineVersion)
	return err
}

func (s *PostgresStore) GetLatestRecommendation(ctx context.Context, userID string) (Recommendation, error) {
	var rec Recommendation
	err := s.pool.QueryRow(ctx, `
		SELECT rec_id, user_id, recommendation_date, input_json, output_json, risk_level, override_json,
		       is_fallback, ai_provider, ai_model, prompt_version, engine_version, created_at
		FROM recommendations
		WHERE user_id=$1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(
		&rec.RecID, &rec.UserID, &rec.RecommendationDate, &rec.InputJSON, &rec.OutputJSON,
		&rec.RiskLevel, &rec.OverrideJSON, &rec.IsFallback, &rec.AIProvider, &rec.AIModel,
		&rec.PromptVersion, &rec.EngineVersion, &rec.CreatedAt,
	)
	if err != nil {
		return Recommendation{}, err
	}
	return rec, nil
}

func (s *PostgresStore) CreateRecommendationFeedback(ctx context.Context, feedback RecommendationFeedback) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO recommendation_feedbacks (feedback_id, rec_id, user_id, useful, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
	`, feedback.FeedbackID, feedback.RecID, feedback.UserID, feedback.Useful, feedback.Reason)
	return err
}

func (s *PostgresStore) HasTrainingConflict(ctx context.Context, userID string, start time.Time, end time.Time, excludeLogID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM training_logs
			WHERE user_id=$1
			  AND deleted_at IS NULL
			  AND ($4 = '' OR log_id <> $4)
			  AND (start_time, start_time + (duration_sec || ' seconds')::interval) OVERLAPS ($2, $3)
			UNION ALL
			SELECT 1
			FROM activities
			WHERE user_id=$1
			  AND (start_time_local, start_time_local + (moving_time_sec || ' seconds')::interval) OVERLAPS ($2, $3)
			LIMIT 1
		)
	`, userID, start, end, excludeLogID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresStore) ListActivities(ctx context.Context, userID string, from time.Time, to time.Time) ([]Activity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, distance_m, moving_time_sec, start_time_local
		FROM activities
		WHERE user_id=$1 AND start_time_local >= $2 AND start_time_local <= $3
		ORDER BY start_time_local DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.UserID, &a.DistanceM, &a.MovingTimeSec, &a.StartTimeLocal); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (s *PostgresStore) ListTrainingLogs(ctx context.Context, userID string, from time.Time, to time.Time) ([]TrainingLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT log_id, user_id, source, training_type, training_type_custom,
		       start_time, duration_sec, distance_km, pace_str, pace_sec_per_km,
		       rpe, discomfort, deleted_at, created_at, updated_at
		FROM training_logs
		WHERE user_id=$1 AND deleted_at IS NULL AND start_time >= $2 AND start_time <= $3
		ORDER BY start_time DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []TrainingLog
	for rows.Next() {
		var log TrainingLog
		if err := rows.Scan(
			&log.LogID, &log.UserID, &log.Source, &log.TrainingType, &log.TrainingTypeCustom,
			&log.StartTime, &log.DurationSec, &log.DistanceKM, &log.PaceStr, &log.PaceSecPerKM,
			&log.RPE, &log.Discomfort, &log.DeletedAt, &log.CreatedAt, &log.UpdatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return logs, nil
}

func (s *PostgresStore) GetTrainingLog(ctx context.Context, logID string) (TrainingLog, error) {
	var log TrainingLog
	err := s.pool.QueryRow(ctx, `
		SELECT log_id, user_id, source, training_type, training_type_custom,
		       start_time, duration_sec, distance_km, pace_str, pace_sec_per_km,
		       rpe, discomfort, deleted_at, created_at, updated_at
		FROM training_logs
		WHERE log_id=$1 AND deleted_at IS NULL
	`, logID).Scan(
		&log.LogID, &log.UserID, &log.Source, &log.TrainingType, &log.TrainingTypeCustom,
		&log.StartTime, &log.DurationSec, &log.DistanceKM, &log.PaceStr, &log.PaceSecPerKM,
		&log.RPE, &log.Discomfort, &log.DeletedAt, &log.CreatedAt, &log.UpdatedAt,
	)
	if err != nil {
		return TrainingLog{}, err
	}
	return log, nil
}

func (s *PostgresStore) UpsertUserProfile(ctx context.Context, p UserProfile) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_profiles (
			user_id, gender, age, height_cm, weight_kg,
			goal_type, goal_cycle, goal_frequency, goal_pace,
			fitness_level, location_lat, location_lng, country,
			province, city, location_source, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW(),NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			gender=EXCLUDED.gender,
			age=EXCLUDED.age,
			height_cm=EXCLUDED.height_cm,
			weight_kg=EXCLUDED.weight_kg,
			goal_type=EXCLUDED.goal_type,
			goal_cycle=EXCLUDED.goal_cycle,
			goal_frequency=EXCLUDED.goal_frequency,
			goal_pace=EXCLUDED.goal_pace,
			fitness_level=EXCLUDED.fitness_level,
			location_lat=EXCLUDED.location_lat,
			location_lng=EXCLUDED.location_lng,
			country=EXCLUDED.country,
			province=EXCLUDED.province,
			city=EXCLUDED.city,
			location_source=EXCLUDED.location_source,
			updated_at=NOW()
	`, p.UserID, p.Gender, p.Age, p.HeightCM, p.WeightKG, p.GoalType, p.GoalCycle, p.GoalFrequency, p.GoalPace, p.FitnessLevel,
		p.LocationLat, p.LocationLng, p.Country, p.Province, p.City, p.LocationSource)
	return err
}

func (s *PostgresStore) GetUserProfile(ctx context.Context, userID string) (UserProfile, error) {
	var p UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, gender, age, height_cm, weight_kg,
		       goal_type, goal_cycle, goal_frequency, goal_pace,
		       fitness_level, location_lat, location_lng, country,
		       province, city, location_source, created_at, updated_at
		FROM user_profiles
		WHERE user_id=$1
	`, userID).Scan(
		&p.UserID, &p.Gender, &p.Age, &p.HeightCM, &p.WeightKG,
		&p.GoalType, &p.GoalCycle, &p.GoalFrequency, &p.GoalPace,
		&p.FitnessLevel, &p.LocationLat, &p.LocationLng, &p.Country,
		&p.Province, &p.City, &p.LocationSource, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return UserProfile{}, err
	}
	return p, nil
}

func (s *PostgresStore) CreateWeatherSnapshot(ctx context.Context, w WeatherSnapshot) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO weather_snapshots (
			user_id, date, temperature_c, feels_like_c, humidity,
			wind_speed_ms, precipitation_prob, aqi, uv_index, risk_level, created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (user_id, date)
		DO UPDATE SET
			temperature_c=EXCLUDED.temperature_c,
			feels_like_c=EXCLUDED.feels_like_c,
			humidity=EXCLUDED.humidity,
			wind_speed_ms=EXCLUDED.wind_speed_ms,
			precipitation_prob=EXCLUDED.precipitation_prob,
			aqi=EXCLUDED.aqi,
			uv_index=EXCLUDED.uv_index,
			risk_level=EXCLUDED.risk_level
	`, w.UserID, w.Date, w.TemperatureC, w.FeelsLikeC, w.Humidity, w.WindSpeedMS, w.PrecipitationProb, w.AQI, w.UVIndex, w.RiskLevel)
	return err
}

func (s *PostgresStore) GetWeatherSnapshot(ctx context.Context, userID string, date time.Time) (WeatherSnapshot, error) {
	var w WeatherSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, date, temperature_c, feels_like_c, humidity,
		       wind_speed_ms, precipitation_prob, aqi, uv_index, risk_level, created_at
		FROM weather_snapshots
		WHERE user_id=$1 AND date=$2
	`, userID, date).Scan(
		&w.UserID, &w.Date, &w.TemperatureC, &w.FeelsLikeC, &w.Humidity,
		&w.WindSpeedMS, &w.PrecipitationProb, &w.AQI, &w.UVIndex, &w.RiskLevel, &w.CreatedAt,
	)
	if err != nil {
		return WeatherSnapshot{}, err
	}
	return w, nil
}
