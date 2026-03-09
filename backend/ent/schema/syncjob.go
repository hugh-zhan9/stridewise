package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// SyncJob tracks async ingestion jobs.
type SyncJob struct {
	ent.Schema
}

func (SyncJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("job_id").Unique(),
		field.String("user_id"),
		field.String("source"),
		field.String("status"),
		field.Int("retry_count").Default(0),
		field.String("error_message").Default(""),
	}
}
