package garmin

import (
	"context"
	"errors"

	"stridewise/backend/internal/connector/common"
	syncjob "stridewise/backend/internal/sync"
)

type Connector struct {
	DataFile string
}

func New(dataFile string) *Connector {
	return &Connector{DataFile: dataFile}
}

func (c *Connector) FetchActivities(_ context.Context, _ string, checkpoint syncjob.Checkpoint) (syncjob.FetchResult, error) {
	if c.DataFile == "" {
		return syncjob.FetchResult{}, errors.New("garmin data_file is empty")
	}
	return common.ParseRunningPageJSON(c.DataFile, checkpoint)
}
