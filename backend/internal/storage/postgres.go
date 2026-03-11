package storage

import (
	"context"
	"database/sql"
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
	UserID                string
	Gender                string
	Age                   int
	HeightCM              int
	WeightKG              int
	RestingHR             int
	GoalType              string
	GoalCycle             string
	GoalFrequency         int
	GoalPace              string
	FitnessLevel          string
	AbilityLevel          string
	AbilityLevelReason    string
	AbilityLevelUpdatedAt *time.Time
	RunningYears          string
	WeeklySessions        string
	WeeklyDistanceKM      string
	LongestRunKM          string
	RecentDiscomfort      string
	LocationLat           float64
	LocationLng           float64
	Country               string
	Province              string
	City                  string
	LocationSource        string
	CreatedAt             time.Time
	UpdatedAt             time.Time
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

type WeatherForecast struct {
	ForecastID       string
	UserID           string
	ForecastDate     time.Time
	TempMaxC         *float64
	TempMinC         *float64
	Humidity         *float64
	PrecipMM         *float64
	PressureHPA      *float64
	VisibilityKM     *float64
	CloudPct         *float64
	UVIndex          *float64
	TextDay          *string
	TextNight        *string
	IconDay          *string
	IconNight        *string
	Wind360Day       *int
	WindDirDay       *string
	WindScaleDay     *string
	WindSpeedDayMS   *float64
	Wind360Night     *int
	WindDirNight     *string
	WindScaleNight   *string
	WindSpeedNightMS *float64
	SunriseTime      *time.Time
	SunsetTime       *time.Time
	MoonriseTime     *time.Time
	MoonsetTime      *time.Time
	MoonPhase        *string
	MoonPhaseIcon    *string
	AQILocal         *int
	AQIQAQI          *int
	AQISource        *string
	CreatedAt        time.Time
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

type TrainingLoadSummary struct {
	Sessions int
	Distance float64
	Duration int
}

type Activity struct {
	ID               int64
	UserID           string
	Source           string
	SourceActivityID string
	Name             string
	DistanceM        float64
	MovingTimeSec    int
	StartTimeLocal   time.Time
}

type BaselineCurrent struct {
	UserID              string
	ComputedAt          time.Time
	DataSessions7d      int
	AcuteLoadSRPE       float64
	ChronicLoadSRPE     float64
	ACWRSRPE            float64
	AcuteLoadDistance   float64
	ChronicLoadDistance float64
	ACWRDistance        float64
	Monotony            float64
	Strain              float64
	PaceAvgSecPerKM     int
	PaceLowSecPerKM     int
	PaceHighSecPerKM    int
	Status              string
}

type BaselineHistory struct {
	BaselineID          string
	UserID              string
	ComputedAt          time.Time
	TriggerType         string
	TriggerRef          string
	DataSessions7d      int
	AcuteLoadSRPE       float64
	ChronicLoadSRPE     float64
	ACWRSRPE            float64
	AcuteLoadDistance   float64
	ChronicLoadDistance float64
	ACWRDistance        float64
	Monotony            float64
	Strain              float64
	PaceAvgSecPerKM     int
	PaceLowSecPerKM     int
	PaceHighSecPerKM    int
	Status              string
}

type NightlyBaselineRun struct {
	RunDate      time.Time
	Status       string
	ErrorMessage string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TrainingSummary struct {
	SummaryID        string
	UserID           string
	SourceType       string
	SourceID         string
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
	SourceType string
	SourceID   string
	LogID      string
	Content    string
	DeletedAt  *time.Time
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

func (s *PostgresStore) UpdateAbilityLevel(ctx context.Context, userID, level, reason string, updatedAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_profiles
		SET ability_level=$2, ability_level_reason=$3, ability_level_updated_at=$4, updated_at=NOW()
		WHERE user_id=$1
	`, userID, level, reason, updatedAt)
	return err
}

func (s *PostgresStore) FindActiveAsyncJob(ctx context.Context, userID, jobType string) (AsyncJob, error) {
	var job AsyncJob
	err := s.pool.QueryRow(ctx, `
		SELECT job_id, job_type, user_id, payload_json, status, retry_count, error_message
		FROM async_jobs
		WHERE user_id=$1 AND job_type=$2 AND status IN ('queued','running')
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, jobType).Scan(
		&job.JobID, &job.JobType, &job.UserID, &job.PayloadJSON, &job.Status, &job.RetryCount, &job.ErrorMessage,
	)
	if err != nil {
		return AsyncJob{}, err
	}
	return job, nil
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

func (s *PostgresStore) GetNightlyBaselineRun(ctx context.Context, runDate time.Time) (NightlyBaselineRun, error) {
	var out NightlyBaselineRun
	err := s.pool.QueryRow(ctx, `
		SELECT run_date, status, error_message, started_at, completed_at, created_at, updated_at
		FROM nightly_baseline_runs
		WHERE run_date=$1
	`, runDate).Scan(
		&out.RunDate, &out.Status, &out.ErrorMessage, &out.StartedAt, &out.CompletedAt, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return NightlyBaselineRun{}, err
	}
	return out, nil
}

func (s *PostgresStore) UpsertNightlyBaselineRun(ctx context.Context, run NightlyBaselineRun) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO nightly_baseline_runs (
			run_date, status, error_message, started_at, completed_at, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,NOW(),NOW())
		ON CONFLICT (run_date)
		DO UPDATE SET
			status=EXCLUDED.status,
			error_message=EXCLUDED.error_message,
			started_at=EXCLUDED.started_at,
			completed_at=EXCLUDED.completed_at,
			updated_at=NOW()
	`, run.RunDate, run.Status, run.ErrorMessage, run.StartedAt, run.CompletedAt)
	return err
}

func (s *PostgresStore) UpsertTrainingSummary(ctx context.Context, summary TrainingSummary) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO training_summaries (
			summary_id, user_id, source_type, source_id, log_id, completion_rate, intensity_match, recovery_advice,
			anomaly_notes, performance_notes, next_suggestion, deleted_at, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		ON CONFLICT (user_id, source_type, source_id)
		DO UPDATE SET
			log_id=EXCLUDED.log_id,
			completion_rate=EXCLUDED.completion_rate,
			intensity_match=EXCLUDED.intensity_match,
			recovery_advice=EXCLUDED.recovery_advice,
			anomaly_notes=EXCLUDED.anomaly_notes,
			performance_notes=EXCLUDED.performance_notes,
			next_suggestion=EXCLUDED.next_suggestion,
			deleted_at=EXCLUDED.deleted_at,
			updated_at=NOW()
	`, summary.SummaryID, summary.UserID, summary.SourceType, summary.SourceID, summary.LogID, summary.CompletionRate, summary.IntensityMatch, summary.RecoveryAdvice,
		summary.AnomalyNotes, summary.PerformanceNotes, summary.NextSuggestion, summary.DeletedAt)
	return err
}

func (s *PostgresStore) GetTrainingSummary(ctx context.Context, logID string) (TrainingSummary, error) {
	return s.GetTrainingSummaryBySource(ctx, "log", logID)
}

func (s *PostgresStore) GetTrainingSummaryBySource(ctx context.Context, sourceType, sourceID string) (TrainingSummary, error) {
	var out TrainingSummary
	err := s.pool.QueryRow(ctx, `
		SELECT summary_id, user_id, source_type, source_id, log_id, completion_rate, intensity_match,
		       recovery_advice, anomaly_notes, performance_notes, next_suggestion,
		       deleted_at, created_at, updated_at
		FROM training_summaries
		WHERE source_type=$1 AND source_id=$2 AND deleted_at IS NULL
	`, sourceType, sourceID).Scan(
		&out.SummaryID, &out.UserID, &out.SourceType, &out.SourceID, &out.LogID, &out.CompletionRate, &out.IntensityMatch,
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
		SELECT summary_id, user_id, source_type, source_id, log_id, completion_rate, intensity_match,
		       recovery_advice, anomaly_notes, performance_notes, next_suggestion,
		       deleted_at, created_at, updated_at, sort_time
		FROM (
			SELECT s.summary_id, s.user_id, s.source_type, s.source_id, s.log_id, s.completion_rate, s.intensity_match,
			       s.recovery_advice, s.anomaly_notes, s.performance_notes, s.next_suggestion,
			       s.deleted_at, s.created_at, s.updated_at, l.start_time AS sort_time
			FROM training_summaries s
			JOIN training_logs l ON s.source_type='log' AND s.source_id = l.log_id
			WHERE s.user_id=$1 AND s.deleted_at IS NULL AND l.deleted_at IS NULL AND l.start_time >= $2 AND l.start_time <= $3
			UNION ALL
			SELECT s.summary_id, s.user_id, s.source_type, s.source_id, s.log_id, s.completion_rate, s.intensity_match,
			       s.recovery_advice, s.anomaly_notes, s.performance_notes, s.next_suggestion,
			       s.deleted_at, s.created_at, s.updated_at, a.start_time_local AS sort_time
			FROM training_summaries s
			JOIN activities a ON s.source_type='activity' AND s.source_id = a.id::text
			WHERE s.user_id=$1 AND s.deleted_at IS NULL AND a.start_time_local >= $2 AND a.start_time_local <= $3
		) summaries
		ORDER BY sort_time DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrainingSummary
	for rows.Next() {
		var s2 TrainingSummary
		var sortTime time.Time
		if err := rows.Scan(
			&s2.SummaryID, &s2.UserID, &s2.SourceType, &s2.SourceID, &s2.LogID, &s2.CompletionRate, &s2.IntensityMatch,
			&s2.RecoveryAdvice, &s2.AnomalyNotes, &s2.PerformanceNotes, &s2.NextSuggestion,
			&s2.DeletedAt, &s2.CreatedAt, &s2.UpdatedAt, &sortTime,
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
		INSERT INTO training_feedbacks (feedback_id, user_id, source_type, source_id, log_id, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW())
	`, feedback.FeedbackID, feedback.UserID, feedback.SourceType, feedback.SourceID, feedback.LogID, feedback.Content)
	return err
}

func (s *PostgresStore) GetLatestTrainingFeedback(ctx context.Context, userID string) (TrainingFeedback, error) {
	var out TrainingFeedback
	err := s.pool.QueryRow(ctx, `
		SELECT feedback_id, user_id, source_type, source_id, log_id, content, deleted_at, created_at
		FROM training_feedbacks
		WHERE user_id=$1 AND deleted_at IS NULL AND content <> ''
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(
		&out.FeedbackID, &out.UserID, &out.SourceType, &out.SourceID, &out.LogID,
		&out.Content, &out.DeletedAt, &out.CreatedAt,
	)
	return out, err
}

func (s *PostgresStore) SoftDeleteTrainingSummaryBySource(ctx context.Context, sourceType, sourceID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE training_summaries
		SET deleted_at=NOW(), updated_at=NOW()
		WHERE source_type=$1 AND source_id=$2 AND deleted_at IS NULL
	`, sourceType, sourceID)
	return err
}

func (s *PostgresStore) SoftDeleteTrainingFeedbackBySource(ctx context.Context, sourceType, sourceID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE training_feedbacks
		SET deleted_at=NOW()
		WHERE source_type=$1 AND source_id=$2 AND deleted_at IS NULL
	`, sourceType, sourceID)
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
		SELECT id, user_id, source, source_activity_id, name, distance_m, moving_time_sec, start_time_local
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
		if err := rows.Scan(&a.ID, &a.UserID, &a.Source, &a.SourceActivityID, &a.Name, &a.DistanceM, &a.MovingTimeSec, &a.StartTimeLocal); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (s *PostgresStore) ListActivitiesBySyncJob(ctx context.Context, jobID string) ([]Activity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.user_id, a.source, a.source_activity_id, a.name, a.distance_m, a.moving_time_sec, a.start_time_local
		FROM activities a
		JOIN raw_activities r ON r.user_id = a.user_id AND r.source = a.source AND r.source_activity_id = a.source_activity_id
		WHERE r.job_id=$1
		ORDER BY a.start_time_local DESC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.UserID, &a.Source, &a.SourceActivityID, &a.Name, &a.DistanceM, &a.MovingTimeSec, &a.StartTimeLocal); err != nil {
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

func (s *PostgresStore) GetRecentTrainingSummary(ctx context.Context, userID string, from time.Time, to time.Time) (TrainingLoadSummary, error) {
	var logDistance float64
	var logDuration int
	var logCount int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(distance_km),0), COALESCE(SUM(duration_sec),0), COUNT(*)
		FROM training_logs
		WHERE user_id=$1 AND deleted_at IS NULL AND start_time >= $2 AND start_time <= $3
	`, userID, from, to).Scan(&logDistance, &logDuration, &logCount)
	if err != nil {
		return TrainingLoadSummary{}, err
	}

	var actDistanceM float64
	var actDuration int
	var actCount int
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(distance_m),0), COALESCE(SUM(moving_time_sec),0), COUNT(*)
		FROM activities
		WHERE user_id=$1 AND start_time_local >= $2 AND start_time_local <= $3
	`, userID, from, to).Scan(&actDistanceM, &actDuration, &actCount)
	if err != nil {
		return TrainingLoadSummary{}, err
	}

	return TrainingLoadSummary{
		Sessions: logCount + actCount,
		Distance: logDistance + actDistanceM/1000.0,
		Duration: logDuration + actDuration,
	}, nil
}

func (s *PostgresStore) GetLatestTrainingDiscomfort(ctx context.Context, userID string) (bool, error) {
	var discomfort bool
	err := s.pool.QueryRow(ctx, `
		SELECT discomfort
		FROM training_logs
		WHERE user_id=$1 AND deleted_at IS NULL
		ORDER BY start_time DESC
		LIMIT 1
	`, userID).Scan(&discomfort)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return discomfort, nil
}

func (s *PostgresStore) ListActiveUsersSince(ctx context.Context, since time.Time) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT user_id
		FROM (
			SELECT user_id
			FROM training_logs
			WHERE deleted_at IS NULL AND start_time >= $1
			UNION
			SELECT user_id
			FROM activities
			WHERE start_time_local >= $1
		) AS active_users
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return users, nil
}

func (s *PostgresStore) UpsertUserProfile(ctx context.Context, p UserProfile) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_profiles (
			user_id, gender, age, height_cm, weight_kg, resting_hr,
			goal_type, goal_cycle, goal_frequency, goal_pace,
			fitness_level, ability_level, ability_level_reason, ability_level_updated_at,
			running_years, weekly_sessions, weekly_distance_km, longest_run_km, recent_discomfort,
			location_lat, location_lng, country,
			province, city, location_source, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,NOW(),NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			gender=EXCLUDED.gender,
			age=EXCLUDED.age,
			height_cm=EXCLUDED.height_cm,
			weight_kg=EXCLUDED.weight_kg,
			resting_hr=EXCLUDED.resting_hr,
			goal_type=EXCLUDED.goal_type,
			goal_cycle=EXCLUDED.goal_cycle,
			goal_frequency=EXCLUDED.goal_frequency,
			goal_pace=EXCLUDED.goal_pace,
			fitness_level=EXCLUDED.fitness_level,
			ability_level=CASE WHEN EXCLUDED.ability_level <> '' THEN EXCLUDED.ability_level ELSE user_profiles.ability_level END,
			ability_level_reason=CASE WHEN EXCLUDED.ability_level <> '' THEN EXCLUDED.ability_level_reason ELSE user_profiles.ability_level_reason END,
			ability_level_updated_at=CASE WHEN EXCLUDED.ability_level <> '' THEN EXCLUDED.ability_level_updated_at ELSE user_profiles.ability_level_updated_at END,
			running_years=EXCLUDED.running_years,
			weekly_sessions=EXCLUDED.weekly_sessions,
			weekly_distance_km=EXCLUDED.weekly_distance_km,
			longest_run_km=EXCLUDED.longest_run_km,
			recent_discomfort=EXCLUDED.recent_discomfort,
			location_lat=EXCLUDED.location_lat,
			location_lng=EXCLUDED.location_lng,
			country=EXCLUDED.country,
			province=EXCLUDED.province,
			city=EXCLUDED.city,
			location_source=EXCLUDED.location_source,
			updated_at=NOW()
	`, p.UserID, p.Gender, p.Age, p.HeightCM, p.WeightKG, p.RestingHR, p.GoalType, p.GoalCycle, p.GoalFrequency, p.GoalPace, p.FitnessLevel,
		p.AbilityLevel, p.AbilityLevelReason, p.AbilityLevelUpdatedAt,
		p.RunningYears, p.WeeklySessions, p.WeeklyDistanceKM, p.LongestRunKM, p.RecentDiscomfort,
		p.LocationLat, p.LocationLng, p.Country, p.Province, p.City, p.LocationSource)
	return err
}

func (s *PostgresStore) GetUserProfile(ctx context.Context, userID string) (UserProfile, error) {
	var p UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, gender, age, height_cm, weight_kg, resting_hr,
		       goal_type, goal_cycle, goal_frequency, goal_pace,
		       fitness_level, ability_level, ability_level_reason, ability_level_updated_at,
		       running_years, weekly_sessions, weekly_distance_km, longest_run_km, recent_discomfort,
		       location_lat, location_lng, country,
		       province, city, location_source, created_at, updated_at
		FROM user_profiles
		WHERE user_id=$1
	`, userID).Scan(
		&p.UserID, &p.Gender, &p.Age, &p.HeightCM, &p.WeightKG, &p.RestingHR,
		&p.GoalType, &p.GoalCycle, &p.GoalFrequency, &p.GoalPace,
		&p.FitnessLevel, &p.AbilityLevel, &p.AbilityLevelReason, &p.AbilityLevelUpdatedAt,
		&p.RunningYears, &p.WeeklySessions, &p.WeeklyDistanceKM, &p.LongestRunKM, &p.RecentDiscomfort,
		&p.LocationLat, &p.LocationLng, &p.Country,
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

func (s *PostgresStore) GetLatestWeatherSnapshot(ctx context.Context, userID string) (WeatherSnapshot, error) {
	var w WeatherSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, date, temperature_c, feels_like_c, humidity,
		       wind_speed_ms, precipitation_prob, aqi, uv_index, risk_level, created_at
		FROM weather_snapshots
		WHERE user_id=$1
		ORDER BY date DESC
		LIMIT 1
	`, userID).Scan(
		&w.UserID, &w.Date, &w.TemperatureC, &w.FeelsLikeC, &w.Humidity,
		&w.WindSpeedMS, &w.PrecipitationProb, &w.AQI, &w.UVIndex, &w.RiskLevel, &w.CreatedAt,
	)
	if err != nil {
		return WeatherSnapshot{}, err
	}
	return w, nil
}

func (s *PostgresStore) UpsertWeatherForecasts(ctx context.Context, forecasts []WeatherForecast) error {
	if len(forecasts) == 0 {
		return nil
	}
	for _, f := range forecasts {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO weather_forecasts (
				forecast_id, user_id, forecast_date,
				temp_max_c, temp_min_c, humidity, precip_mm, pressure_hpa, visibility_km,
				cloud_pct, uv_index, aqi_local, aqi_qaqi, aqi_source, text_day, text_night, icon_day, icon_night,
				wind360_day, wind_dir_day, wind_scale_day, wind_speed_day_ms,
				wind360_night, wind_dir_night, wind_scale_night, wind_speed_night_ms,
				sunrise_time, sunset_time, moonrise_time, moonset_time, moon_phase, moon_phase_icon,
				created_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,NOW()
			)
			ON CONFLICT (user_id, forecast_date)
			DO UPDATE SET
				forecast_id=EXCLUDED.forecast_id,
				temp_max_c=EXCLUDED.temp_max_c,
				temp_min_c=EXCLUDED.temp_min_c,
				humidity=EXCLUDED.humidity,
				precip_mm=EXCLUDED.precip_mm,
				pressure_hpa=EXCLUDED.pressure_hpa,
				visibility_km=EXCLUDED.visibility_km,
				cloud_pct=EXCLUDED.cloud_pct,
				uv_index=EXCLUDED.uv_index,
				aqi_local=EXCLUDED.aqi_local,
				aqi_qaqi=EXCLUDED.aqi_qaqi,
				aqi_source=EXCLUDED.aqi_source,
				text_day=EXCLUDED.text_day,
				text_night=EXCLUDED.text_night,
				icon_day=EXCLUDED.icon_day,
				icon_night=EXCLUDED.icon_night,
				wind360_day=EXCLUDED.wind360_day,
				wind_dir_day=EXCLUDED.wind_dir_day,
				wind_scale_day=EXCLUDED.wind_scale_day,
				wind_speed_day_ms=EXCLUDED.wind_speed_day_ms,
				wind360_night=EXCLUDED.wind360_night,
				wind_dir_night=EXCLUDED.wind_dir_night,
				wind_scale_night=EXCLUDED.wind_scale_night,
				wind_speed_night_ms=EXCLUDED.wind_speed_night_ms,
				sunrise_time=EXCLUDED.sunrise_time,
				sunset_time=EXCLUDED.sunset_time,
				moonrise_time=EXCLUDED.moonrise_time,
				moonset_time=EXCLUDED.moonset_time,
				moon_phase=EXCLUDED.moon_phase,
				moon_phase_icon=EXCLUDED.moon_phase_icon
		`, f.ForecastID, f.UserID, f.ForecastDate, f.TempMaxC, f.TempMinC, f.Humidity, f.PrecipMM, f.PressureHPA,
			f.VisibilityKM, f.CloudPct, f.UVIndex, f.AQILocal, f.AQIQAQI, f.AQISource, f.TextDay, f.TextNight, f.IconDay, f.IconNight,
			f.Wind360Day, f.WindDirDay, f.WindScaleDay, f.WindSpeedDayMS,
			f.Wind360Night, f.WindDirNight, f.WindScaleNight, f.WindSpeedNightMS,
			f.SunriseTime, f.SunsetTime, f.MoonriseTime, f.MoonsetTime, f.MoonPhase, f.MoonPhaseIcon,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) GetWeatherForecasts(ctx context.Context, userID string, from time.Time, to time.Time) ([]WeatherForecast, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT forecast_id, user_id, forecast_date,
		       temp_max_c, temp_min_c, humidity, precip_mm, pressure_hpa, visibility_km,
		       cloud_pct, uv_index, aqi_local, aqi_qaqi, aqi_source, text_day, text_night, icon_day, icon_night,
		       wind360_day, wind_dir_day, wind_scale_day, wind_speed_day_ms,
		       wind360_night, wind_dir_night, wind_scale_night, wind_speed_night_ms,
		       sunrise_time, sunset_time, moonrise_time, moonset_time, moon_phase, moon_phase_icon,
		       created_at
		FROM weather_forecasts
		WHERE user_id=$1 AND forecast_date BETWEEN $2 AND $3
		ORDER BY forecast_date ASC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WeatherForecast
	for rows.Next() {
		var f WeatherForecast
		var tempMax, tempMin, humidity, precip, pressure, visibility, cloud, uv sql.NullFloat64
		var aqiLocal, aqiQAQI sql.NullInt32
		var aqiSource sql.NullString
		var textDay, textNight, iconDay, iconNight sql.NullString
		var wind360Day, wind360Night sql.NullInt32
		var windDirDay, windScaleDay, windDirNight, windScaleNight sql.NullString
		var windSpeedDay, windSpeedNight sql.NullFloat64
		var sunrise, sunset, moonrise, moonset sql.NullTime
		var moonPhase, moonPhaseIcon sql.NullString

		if err := rows.Scan(
			&f.ForecastID, &f.UserID, &f.ForecastDate,
			&tempMax, &tempMin, &humidity, &precip, &pressure, &visibility,
			&cloud, &uv, &aqiLocal, &aqiQAQI, &aqiSource, &textDay, &textNight, &iconDay, &iconNight,
			&wind360Day, &windDirDay, &windScaleDay, &windSpeedDay,
			&wind360Night, &windDirNight, &windScaleNight, &windSpeedNight,
			&sunrise, &sunset, &moonrise, &moonset, &moonPhase, &moonPhaseIcon,
			&f.CreatedAt,
		); err != nil {
			return nil, err
		}

		f.TempMaxC = nullFloatPtr(tempMax)
		f.TempMinC = nullFloatPtr(tempMin)
		f.Humidity = nullFloatPtr(humidity)
		f.PrecipMM = nullFloatPtr(precip)
		f.PressureHPA = nullFloatPtr(pressure)
		f.VisibilityKM = nullFloatPtr(visibility)
		f.CloudPct = nullFloatPtr(cloud)
		f.UVIndex = nullFloatPtr(uv)
		f.AQILocal = nullIntPtr(aqiLocal)
		f.AQIQAQI = nullIntPtr(aqiQAQI)
		f.AQISource = nullStringPtr(aqiSource)
		f.TextDay = nullStringPtr(textDay)
		f.TextNight = nullStringPtr(textNight)
		f.IconDay = nullStringPtr(iconDay)
		f.IconNight = nullStringPtr(iconNight)
		f.Wind360Day = nullIntPtr(wind360Day)
		f.WindDirDay = nullStringPtr(windDirDay)
		f.WindScaleDay = nullStringPtr(windScaleDay)
		f.WindSpeedDayMS = nullFloatPtr(windSpeedDay)
		f.Wind360Night = nullIntPtr(wind360Night)
		f.WindDirNight = nullStringPtr(windDirNight)
		f.WindScaleNight = nullStringPtr(windScaleNight)
		f.WindSpeedNightMS = nullFloatPtr(windSpeedNight)
		f.SunriseTime = nullTimePtr(sunrise)
		f.SunsetTime = nullTimePtr(sunset)
		f.MoonriseTime = nullTimePtr(moonrise)
		f.MoonsetTime = nullTimePtr(moonset)
		f.MoonPhase = nullStringPtr(moonPhase)
		f.MoonPhaseIcon = nullStringPtr(moonPhaseIcon)

		out = append(out, f)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func nullFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	val := v.Float64
	return &val
}

func nullIntPtr(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}
	val := int(v.Int32)
	return &val
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	val := v.String
	return &val
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	val := v.Time
	return &val
}
