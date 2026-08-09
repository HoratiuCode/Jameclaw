package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestUserFacingProcessingErrorExplainsUnavailableModel(t *testing.T) {
	err := errors.New(`API request failed: Status: 404 Body: {"message":"Model 'default' not found. The requested model does not exist"}`)
	message := userFacingProcessingError("jame", err)

	if !strings.Contains(message, "selected model is not available") {
		t.Fatalf("message = %q, want unavailable-model guidance", message)
	}
	if !strings.Contains(message, "Settings → AI Provider") {
		t.Fatalf("message = %q, want Settings recovery path", message)
	}
}

func TestUserFacingProcessingErrorExplainsInsufficientCredits(t *testing.T) {
	err := errors.New(`API request failed: Status: 404 Body: {"message":"Model requires available credits. Your account balance is too low"}`)
	message := userFacingProcessingError("jame", err)

	if !strings.Contains(message, "does not have enough credits") {
		t.Fatalf("message = %q, want insufficient-credit guidance", message)
	}
	if !strings.Contains(message, "Settings → AI Provider") {
		t.Fatalf("message = %q, want Settings recovery path", message)
	}
}
