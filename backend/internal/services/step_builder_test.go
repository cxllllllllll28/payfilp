package services

import (
	"testing"
)

func TestStepBuilder_ApproveSwapStake(t *testing.T) {
	t.Skip("requires TxManager + client — integration test, tested manually with TestExecuteSwapIntent")
}

func TestStepBuilder_AmountToBig(t *testing.T) {
	a := amountToBig("50")
	if a.String() != "50000000" {
		t.Errorf("expected 50000000, got %s", a.String())
	}
}
