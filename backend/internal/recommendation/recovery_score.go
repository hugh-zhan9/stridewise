package recommendation

import "math"

type RecoveryScore struct {
	OverallScore      float64
	FatigueScore      float64
	RecoveryScore     float64
	ACWRComponent     float64
	MonotonyComponent float64
	StrainComponent   float64
	DiscomfortPenalty float64
	RestingHRPenalty  float64
	RecoveryStatus    string
}

func BuildRecoveryScore(acwr float64, monotony float64, strain float64, hasDiscomfort bool, restingHR int) RecoveryScore {
	acwrRisk := clampFloat((acwr-1.0)/0.7*100, 0, 100)
	monotonyRisk := clampFloat((monotony-1.0)/1.4*100, 0, 100)
	strainRisk := clampFloat((strain-250)/500*100, 0, 100)

	discomfortPenalty := 0.0
	if hasDiscomfort {
		discomfortPenalty = 15.0
	}
	restingPenalty := 0.0
	if restingHR > 65 {
		restingPenalty = clampFloat(float64(restingHR-65)*0.8, 0, 12)
	}

	fatigue := clampFloat(acwrRisk*0.45+monotonyRisk*0.25+strainRisk*0.30+discomfortPenalty+restingPenalty, 0, 100)
	recovery := clampFloat(100-fatigue, 0, 100)
	overall := recovery

	status := "green"
	if acwr >= 1.55 || monotony >= 2.2 || overall < 40 {
		status = "red"
	} else if acwr >= 1.35 || monotony >= 2.0 || overall < 65 {
		status = "yellow"
	}

	return RecoveryScore{
		OverallScore:      round2(overall),
		FatigueScore:      round2(fatigue),
		RecoveryScore:     round2(recovery),
		ACWRComponent:     round2(acwrRisk),
		MonotonyComponent: round2(monotonyRisk),
		StrainComponent:   round2(strainRisk),
		DiscomfortPenalty: round2(discomfortPenalty),
		RestingHRPenalty:  round2(restingPenalty),
		RecoveryStatus:    status,
	}
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
