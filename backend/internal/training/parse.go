package training

import (
	"errors"
	"strconv"
	"strings"
)

var allowedTypes = map[string]struct{}{
	"轻松跑": {},
	"有氧跑": {},
	"间歇跑": {},
	"长距离": {},
}

func ParseDuration(input string) (int, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 3 {
		return 0, errors.New("duration format must be HH:MM:SS")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 {
		return 0, errors.New("duration hours invalid")
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m >= 60 {
		return 0, errors.New("duration minutes invalid")
	}
	s, err := strconv.Atoi(parts[2])
	if err != nil || s < 0 || s >= 60 {
		return 0, errors.New("duration seconds invalid")
	}
	return h*3600 + m*60 + s, nil
}

func ParsePace(input string) (int, error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimSuffix(trimmed, "''")
	parts := strings.Split(trimmed, "'")
	if len(parts) != 2 {
		return 0, errors.New("pace format must be mm'ss''")
	}
	m, err := strconv.Atoi(parts[0])
	if err != nil || m <= 0 {
		return 0, errors.New("pace minutes invalid")
	}
	s, err := strconv.Atoi(parts[1])
	if err != nil || s < 0 || s >= 60 {
		return 0, errors.New("pace seconds invalid")
	}
	return m*60 + s, nil
}

func NormalizeTrainingType(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errors.New("training_type required")
	}
	if _, ok := allowedTypes[input]; ok {
		return input, "", nil
	}
	return "custom", input, nil
}
