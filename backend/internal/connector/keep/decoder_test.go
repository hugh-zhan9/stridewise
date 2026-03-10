package keep

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeRunmapData_RoundTrip(t *testing.T) {
	points := []map[string]any{{"latitude": 30.0, "longitude": 120.0, "timestamp": 1}}
	payload, err := encodeRunmapForTest(points)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := decodeRunmapData(payload, false)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
}

func encodeRunmapForTest(points []map[string]any) (string, error) {
	raw, err := json.Marshal(points)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		_ = gw.Close()
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
