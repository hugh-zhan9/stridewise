package keep

import (
	"context"

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
	return common.ParseRunningPageJSON(c.DataFile, checkpoint)
}
