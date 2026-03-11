package server

import "context"

type abilityEnqueuerStub struct {
	called bool
}

func (a *abilityEnqueuerStub) EnqueueAbilityLevelCalc(_ context.Context, _ string, _ string, _ string) (string, error) {
	a.called = true
	return "job-ability-1", nil
}
