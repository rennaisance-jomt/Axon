package types

import (
	"encoding/json"
	"testing"
)

func TestAPIError_JSON(t *testing.T) {
	errObj := APIError{
		Error:       true,
		ErrorType:   ErrSessionNotFound,
		Message:     "Session not found",
		Suggestion:  "Create a new session",
		Recoverable: false,
	}

	data, err := json.Marshal(errObj)
	if err != nil {
		t.Fatalf("Failed to marshal APIError: %v", err)
	}

	var parsed APIError
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	if parsed.ErrorType != ErrSessionNotFound || parsed.Message != "Session not found" {
		t.Fatalf("APIError JSON serialization mismatch")
	}
}

func TestActionRequest_Structure(t *testing.T) {
	req := ActionRequest{
		Ref:     "e1",
		Action:  ActionClick,
		Value:   "",
		Confirm: true,
	}

	if req.Action != "click" {
		t.Errorf("Expected action 'click', got %s", req.Action)
	}
	if req.Confirm != true {
		t.Errorf("Expected confirm=true")
	}
}
