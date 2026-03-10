package keep

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	syncjob "stridewise/backend/internal/sync"
)

const keepBaseURL = "https://api.gotokeep.com"

var keepSportTypes = []string{"running", "hiking", "cycling"}

var keepTypeMap = map[string]string{
	"outdoorWalking": "Walk",
	"outdoorRunning": "Run",
	"outdoorCycling": "Ride",
	"indoorRunning":  "VirtualRun",
	"mountaineering": "Hiking",
}

type Connector struct {
	phone      string
	password   string
	baseURL    string
	httpClient *http.Client
	client     *KeepClient
	sleepFn    func(time.Duration)
}

func NewLive(phone, password, baseURL string, httpClient *http.Client) *Connector {
	if baseURL == "" {
		baseURL = keepBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Connector{
		phone:      phone,
		password:   password,
		baseURL:    baseURL,
		httpClient: httpClient,
		sleepFn:    time.Sleep,
	}
}

func (c *Connector) FetchActivities(ctx context.Context, _ string, checkpoint syncjob.Checkpoint) (syncjob.FetchResult, error) {
	if c.phone == "" || c.password == "" {
		return syncjob.FetchResult{}, errors.New("keep credential is empty")
	}
	if c.client == nil {
		c.client = NewKeepClient(c.baseURL, c.httpClient)
	}

	token, err := c.client.Login(ctx, c.phone, c.password)
	if err != nil {
		return syncjob.FetchResult{}, err
	}

	checkpointMs := checkpoint.LastSyncedAt.UnixMilli()
	lastSyncedAt := checkpoint.LastSyncedAt
	activities := make([]syncjob.RawActivity, 0)

	for _, sportType := range keepSportTypes {
		lastDate := int64(0)
		for {
			ids, lastTimestamp, err := c.client.FetchRunIDs(ctx, token, sportType, lastDate)
			if err != nil {
				return syncjob.FetchResult{}, err
			}
			for _, id := range ids {
				detail, err := c.client.FetchRunDetail(ctx, token, sportType, id)
				if err != nil {
					continue
				}
				activity, startUTC, ok := parseKeepRunData(detail)
				if !ok {
					continue
				}
				if !checkpoint.LastSyncedAt.IsZero() && !startUTC.After(checkpoint.LastSyncedAt) {
					continue
				}
				activities = append(activities, activity)
				if startUTC.After(lastSyncedAt) {
					lastSyncedAt = startUTC
				}
			}
			if lastTimestamp == 0 || (checkpointMs > 0 && lastTimestamp <= checkpointMs) {
				break
			}
			lastDate = lastTimestamp
			if c.sleepFn != nil {
				c.sleepFn(1 * time.Second)
			}
		}
	}

	return syncjob.FetchResult{
		Activities:   activities,
		LastSyncedAt: lastSyncedAt,
	}, nil
}

type keepRunDetail struct {
	Data struct {
		ID        string  `json:"id"`
		StartTime int64   `json:"startTime"`
		EndTime   int64   `json:"endTime"`
		Duration  int     `json:"duration"`
		Distance  float64 `json:"distance"`
		DataType  string  `json:"dataType"`
		Timezone  string  `json:"timezone"`
		GeoPoints any     `json:"geoPoints"`
		HeartRate any     `json:"heartRate"`
	} `json:"data"`
}

func parseKeepRunData(detail keepRunDetail) (syncjob.RawActivity, time.Time, bool) {
	data := detail.Data
	if data.Duration == 0 {
		return syncjob.RawActivity{}, time.Time{}, false
	}
	startUTC := time.UnixMilli(data.StartTime).UTC()
	startLocal := adjustTime(startUTC, data.Timezone)
	activityType := keepTypeName(data.DataType)
	name := fmt.Sprintf("%s from keep", activityType)
	keepID := extractKeepID(data.ID)

	raw := map[string]any{
		"run_id":            keepID,
		"name":              name,
		"distance":          data.Distance,
		"moving_time":       data.Duration,
		"start_date":        startUTC.Format("2006-01-02 15:04:05"),
		"start_date_local":  startLocal.Format("2006-01-02 15:04:05"),
		"data_type":         data.DataType,
		"summary_polyline":  "",
		"average_heartrate": nil,
		"elevation_gain":    nil,
	}

	return syncjob.RawActivity{
		SourceActivityID: keepID,
		Name:             name,
		DistanceM:        data.Distance,
		MovingTimeSec:    data.Duration,
		StartTime:        startLocal,
		SummaryPolyline:  "",
		Raw:              raw,
	}, startUTC, true
}

func keepTypeName(dataType string) string {
	if v, ok := keepTypeMap[dataType]; ok {
		return v
	}
	return dataType
}

func extractKeepID(v string) string {
	parts := strings.Split(v, "_")
	if len(parts) >= 2 && parts[1] != "" {
		return parts[1]
	}
	return v
}

func adjustTime(t time.Time, tzName string) time.Time {
	if tzName == "" {
		return t
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return t
	}
	return t.In(loc)
}
