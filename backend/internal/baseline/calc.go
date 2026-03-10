package baseline

import "math"

type SessionInput struct {
	DurationMin   float64
	DistanceKM    float64
	RPE           *int
	PaceSecPerKM  int
	StartDayIndex int
}

type Metrics struct {
	DataSessions7d   int
	AcuteSRPE        float64
	ChronicSRPE      float64
	ACWRSRPE         float64
	AcuteDistance    float64
	ChronicDistance  float64
	ACWRDistance     float64
	Monotony         float64
	Strain           float64
	PaceAvgSecPerKM  int
	PaceLowSecPerKM  int
	PaceHighSecPerKM int
	Status           string
}

func CalcMetrics(items []SessionInput, sessions7d int) Metrics {
	var acuteSRPE float64
	var chronicSRPE float64
	var acuteDistance float64
	var chronicDistance float64
	var paceAvg int

	var dailySRPE [7]float64
	var dailyDistance [7]float64
	var hasSRPE bool

	for _, s := range items {
		if s.StartDayIndex < 0 || s.StartDayIndex > 27 {
			continue
		}
		if s.DistanceKM > 0 {
			chronicDistance += s.DistanceKM
			if s.StartDayIndex < 7 {
				acuteDistance += s.DistanceKM
				dailyDistance[s.StartDayIndex] += s.DistanceKM
			}
		}
		if s.RPE != nil && s.DurationMin > 0 {
			load := s.DurationMin * float64(*s.RPE)
			chronicSRPE += load
			if s.StartDayIndex < 7 {
				acuteSRPE += load
				dailySRPE[s.StartDayIndex] += load
				hasSRPE = true
			}
		}
	}

	var dailyForMonotony [7]float64
	if hasSRPE {
		dailyForMonotony = dailySRPE
	} else {
		dailyForMonotony = dailyDistance
	}

	monotony := calcMonotony(dailyForMonotony[:])
	strain := 0.0
	if hasSRPE {
		strain = acuteSRPE * monotony
	} else {
		strain = acuteDistance * monotony
	}

	chronicSRPE = chronicSRPE / 4
	chronicDistance = chronicDistance / 4

	acwrSRPE := 0.0
	if chronicSRPE > 0 {
		acwrSRPE = acuteSRPE / chronicSRPE
	}
	acwrDistance := 0.0
	if chronicDistance > 0 {
		acwrDistance = acuteDistance / chronicDistance
	}

	paceAvg = CalcPaceAverage(items)
	paceLow := 0
	paceHigh := 0
	if paceAvg > 0 {
		paceLow = int(math.Round(float64(paceAvg) * 0.9))
		paceHigh = int(math.Round(float64(paceAvg) * 1.1))
	}

	status := "ok"
	if sessions7d < 3 {
		status = "insufficient_data"
	}

	return Metrics{
		DataSessions7d:   sessions7d,
		AcuteSRPE:        acuteSRPE,
		ChronicSRPE:      chronicSRPE,
		ACWRSRPE:         acwrSRPE,
		AcuteDistance:    acuteDistance,
		ChronicDistance:  chronicDistance,
		ACWRDistance:     acwrDistance,
		Monotony:         monotony,
		Strain:           strain,
		PaceAvgSecPerKM:  paceAvg,
		PaceLowSecPerKM:  paceLow,
		PaceHighSecPerKM: paceHigh,
		Status:           status,
	}
}

func CalcPaceAverage(items []SessionInput) int {
	var totalDist float64
	var weighted float64
	for _, s := range items {
		if s.DistanceKM <= 0 || s.PaceSecPerKM <= 0 {
			continue
		}
		totalDist += s.DistanceKM
		weighted += s.DistanceKM * float64(s.PaceSecPerKM)
	}
	if totalDist == 0 {
		return 0
	}
	return int(math.Round(weighted / totalDist))
}

func calcMonotony(daily []float64) float64 {
	if len(daily) == 0 {
		return 0
	}
	var sum float64
	for _, v := range daily {
		sum += v
	}
	mean := sum / float64(len(daily))
	var variance float64
	for _, v := range daily {
		diff := v - mean
		variance += diff * diff
	}
	variance = variance / float64(len(daily))
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}
	return mean / std
}
