package ai

import (
	"context"
	"testing"
)

func TestOpenAISummarizerRequiresConfig(t *testing.T) {
	s := NewOpenAISummarizer(OpenAIConfig{})
	_, err := s.Summarize(context.Background(), SummaryInput{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
