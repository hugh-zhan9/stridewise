package task

import (
	"encoding/json"
	"errors"
)

const TypeSyncJob = "sync:job"

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
	UserID string `json:"user_id"`
	Source string `json:"source"`
}

func EncodeSyncJobPayload(p SyncJobPayload) ([]byte, error) {
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
	if p.UserID == "" {
		return SyncJobPayload{}, errors.New("user_id is required")
	}
	if _, ok := allowedSources[p.Source]; !ok {
		return SyncJobPayload{}, errors.New("unsupported source")
	}
	return p, nil
}
