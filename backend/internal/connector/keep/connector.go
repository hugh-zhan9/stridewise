package keep

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	syncjob "stridewise/backend/internal/sync"
)

type Connector struct {
	DataFile string
}

type runningPageActivity struct {
	RunID           any     `json:"run_id"`
	Name            string  `json:"name"`
	Distance        float64 `json:"distance"`
	MovingTime      string  `json:"moving_time"`
	StartDateLocal  string  `json:"start_date_local"`
	SummaryPolyline string  `json:"summary_polyline"`
}

func New(dataFile string) *Connector {
	return &Connector{DataFile: dataFile}
}

func (c *Connector) FetchActivities(_ context.Context, _ string, checkpoint syncjob.Checkpoint) (syncjob.FetchResult, error) {
	if c.DataFile == "" {
		return syncjob.FetchResult{}, errors.New("keep data_file is empty")
	}
	b, err := os.ReadFile(c.DataFile)
	if err != nil {
		return syncjob.FetchResult{}, err
	}

	var list []runningPageActivity
	if err := json.Unmarshal(b, &list); err != nil {
		return syncjob.FetchResult{}, err
	}

	out := make([]syncjob.RawActivity, 0, len(list))
	lastSyncedAt := checkpoint.LastSyncedAt
	for _, item := range list {
		start, err := parseStartDate(item.StartDateLocal)
		if err != nil {
			continue
		}
		if !checkpoint.LastSyncedAt.IsZero() && !start.After(checkpoint.LastSyncedAt) {
			continue
		}
		moving := parseMovingTime(item.MovingTime)
		sourceID := toSourceID(item.RunID)
		if sourceID == "" {
			continue
		}
		out = append(out, syncjob.RawActivity{
			SourceActivityID: sourceID,
			Name:             item.Name,
			DistanceM:        item.Distance,
			MovingTimeSec:    moving,
			StartTime:        start,
			SummaryPolyline:  item.SummaryPolyline,
			Raw: map[string]any{
				"run_id":           item.RunID,
				"name":             item.Name,
				"distance":         item.Distance,
				"moving_time":      item.MovingTime,
				"start_date_local": item.StartDateLocal,
			},
		})
		if start.After(lastSyncedAt) {
			lastSyncedAt = start
		}
	}
	return syncjob.FetchResult{
		Activities:   out,
		LastSyncedAt: lastSyncedAt,
	}, nil
}

func parseStartDate(s string) (time.Time, error) {
	layouts := []string{"2006-01-02 15:04:05", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid start date")
}

// running_page 里 moving_time 可能是 "2 days, 12:34:56" 或 "12:34:56"
func parseMovingTime(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ", ")
	days := 0
	timePart := parts[len(parts)-1]
	if len(parts) == 2 {
		d := strings.Fields(parts[0])
		if len(d) > 0 {
			days, _ = strconv.Atoi(d[0])
		}
	}
	hms := strings.Split(timePart, ":")
	if len(hms) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(hms[0])
	m, _ := strconv.Atoi(hms[1])
	sec, _ := strconv.Atoi(hms[2])
	return ((days*24+h)*60+m)*60 + sec
}

func toSourceID(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int:
		return strconv.Itoa(x)
	default:
		return ""
	}
}
