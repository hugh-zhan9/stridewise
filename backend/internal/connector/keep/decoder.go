package keep

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
)

const (
	keepAESKey = "NTZmZTU5OzgyZzpkODczYw=="
	keepAESIV  = "MjM0Njg5MjQzMjkyMDMwMA=="
)

func decodeRunmapData(text string, isGeo bool) ([]map[string]any, error) {
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	if isGeo {
		key, err := base64.StdEncoding.DecodeString(keepAESKey)
		if err != nil {
			return nil, err
		}
		iv, err := base64.StdEncoding.DecodeString(keepAESIV)
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		if len(raw)%block.BlockSize() != 0 {
			return nil, errors.New("keep encrypted payload size invalid")
		}
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(raw, raw)
		raw = pkcs7Unpad(raw, block.BlockSize())
	}

	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	decoded, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	if err := json.Unmarshal(decoded, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func pkcs7Unpad(in []byte, blockSize int) []byte {
	if len(in) == 0 || len(in)%blockSize != 0 {
		return in
	}
	pad := int(in[len(in)-1])
	if pad == 0 || pad > blockSize || pad > len(in) {
		return in
	}
	for i := len(in) - pad; i < len(in); i++ {
		if int(in[i]) != pad {
			return in
		}
	}
	return in[:len(in)-pad]
}

func gcj2wgs(lat, lng float64) (float64, float64) {
	if outOfChina(lat, lng) {
		return lat, lng
	}
	dLat := transformLat(lng-105.0, lat-35.0)
	dLng := transformLng(lng-105.0, lat-35.0)
	radLat := lat / 180.0 * math.Pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * math.Pi)
	dLng = (dLng * 180.0) / (a / sqrtMagic * math.Cos(radLat) * math.Pi)
	mgLat := lat + dLat
	mgLng := lng + dLng
	return lat*2 - mgLat, lng*2 - mgLng
}

const (
	a  = 6378245.0
	ee = 0.00669342162296594323
)

func outOfChina(lat, lng float64) bool {
	return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271
}

func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*math.Pi) + 40.0*math.Sin(y/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*math.Pi) + 320*math.Sin(y*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLng(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*math.Pi) + 40.0*math.Sin(x/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*math.Pi) + 300.0*math.Sin(x/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}

func encodePolyline(points [][2]float64) string {
	var result strings.Builder
	prevLat := 0
	prevLng := 0
	for _, p := range points {
		lat := int(math.Round(p[0] * 1e5))
		lng := int(math.Round(p[1] * 1e5))
		result.WriteString(encodePolylineValue(lat - prevLat))
		result.WriteString(encodePolylineValue(lng - prevLng))
		prevLat = lat
		prevLng = lng
	}
	return result.String()
}

func encodePolylineValue(value int) string {
	value <<= 1
	if value < 0 {
		value = ^value
	}
	var out []byte
	for value >= 0x20 {
		out = append(out, byte((0x20|(value&0x1f))+63))
		value >>= 5
	}
	out = append(out, byte(value+63))
	return string(out)
}
