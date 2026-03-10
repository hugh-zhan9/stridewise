package task

import (
	"encoding/json"
	"errors"
)

const TypeSyncJob = "sync:job"
const TypeTrainingRecalc = "training:recalc"

var allowedSources = map[string]struct{}{
	"keep":   {},
	"strava": {},
	"garmin": {},
	"nike":   {},
	"gpx":    {},
	"tcx":    {},
	"fit":    {},
}

type SyncJobPayload struct {
	JobID      string `json:"job_id"`
	UserID     string `json:"user_id"`
	Source     string `json:"source"`
	RetryCount int    `json:"retry_count"`
}

type TrainingRecalcPayload struct {
	JobID     string `json:"job_id"`
	UserID    string `json:"user_id"`
	LogID     string `json:"log_id"`
	Operation string `json:"operation"`
}

func EncodeSyncJobPayload(p SyncJobPayload) ([]byte, error) {
	if p.JobID == "" {
		return nil, errors.New("job_id is required")
	}
	if p.UserID == "" {
		return nil, errors.New("user_id is required")
	}
	if _, ok := allowedSources[p.Source]; !ok {
		return nil, errors.New("unsupported source")
	}
	return json.Marshal(p)
}

func DecodeSyncJobPayload(b []byte) (SyncJobPayload, error) {
	var p SyncJobPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return SyncJobPayload{}, err
	}
	if p.JobID == "" {
		return SyncJobPayload{}, errors.New("job_id is required")
	}
	if p.UserID == "" {
		return SyncJobPayload{}, errors.New("user_id is required")
	}
	if _, ok := allowedSources[p.Source]; !ok {
		return SyncJobPayload{}, errors.New("unsupported source")
	}
	if p.RetryCount < 0 {
		return SyncJobPayload{}, errors.New("retry_count cannot be negative")
	}
	return p, nil
}

func EncodeTrainingRecalcPayload(p TrainingRecalcPayload) ([]byte, error) {
	if p.JobID == "" {
		return nil, errors.New("job_id is required")
	}
	if p.UserID == "" {
		return nil, errors.New("user_id is required")
	}
	if p.LogID == "" {
		return nil, errors.New("log_id is required")
	}
	if p.Operation != "create" && p.Operation != "update" && p.Operation != "delete" {
		return nil, errors.New("operation invalid")
	}
	return json.Marshal(p)
}

func DecodeTrainingRecalcPayload(b []byte) (TrainingRecalcPayload, error) {
	var p TrainingRecalcPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return TrainingRecalcPayload{}, err
	}
	if p.JobID == "" {
		return TrainingRecalcPayload{}, errors.New("job_id is required")
	}
	if p.UserID == "" {
		return TrainingRecalcPayload{}, errors.New("user_id is required")
	}
	if p.LogID == "" {
		return TrainingRecalcPayload{}, errors.New("log_id is required")
	}
	if p.Operation != "create" && p.Operation != "update" && p.Operation != "delete" {
		return TrainingRecalcPayload{}, errors.New("operation invalid")
	}
	return p, nil
}
