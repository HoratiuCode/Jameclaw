package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/tools"
)

func TestRecoverableToolFailureClassification(t *testing.T) {
	if !recoverableToolFailure(tools.ErrorResult("file not found")) {
		t.Fatal("file-not-found should be recoverable")
	}
	if recoverableToolFailure(tools.ErrorResult("permission denied")) {
		t.Fatal("permission errors should require user action")
	}
}

func TestRecoveryBudgetStopsAfterTwoAttempts(t *testing.T) {
	ts := &turnState{}
	first, canRecover := ts.recordRecoveryFailure(tools.ErrorResult("connection timeout"))
	if first != 1 || !canRecover {
		t.Fatalf("first recovery = %d, %v", first, canRecover)
	}
	second, canRecover := ts.recordRecoveryFailure(tools.ErrorResult("connection timeout"))
	if second != 2 || canRecover {
		t.Fatalf("second recovery = %d, %v", second, canRecover)
	}
	message := recoveryInstruction("web", tools.ErrorResult("connection timeout"), second, canRecover)
	if !strings.Contains(message, "Do not call another tool") {
		t.Fatalf("unexpected final recovery instruction: %s", message)
	}
}
