package recommendation

func CalcRecoveryStatus(acwr float64, monotony float64) string {
	if acwr >= 2.0 || acwr > 1.5 || monotony >= 2.2 {
		return "red"
	}
	if acwr > 1.3 || monotony >= 2.0 {
		return "yellow"
	}
	return "green"
}
