package keep

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
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

func encodeRunmapForTestGeo(points []map[string]any) (string, error) {
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
	key, err := base64.StdEncoding.DecodeString(keepAESKey)
	if err != nil {
		return "", err
	}
	iv, err := base64.StdEncoding.DecodeString(keepAESIV)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := pkcs7Pad(buf.Bytes(), block.BlockSize())
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func pkcs7Pad(in []byte, blockSize int) []byte {
	pad := blockSize - (len(in) % blockSize)
	out := make([]byte, len(in)+pad)
	copy(out, in)
	for i := len(in); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}
