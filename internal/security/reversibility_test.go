package security

import "testing"

func TestActionClassifier_ClassifyAction(t *testing.T) {
	classifier := NewActionClassifier()

	tests := []struct {
		name          string
		actionType    string
		elementLabel  string
		elementType   string
		wantReversibility string
	}{
		{
			name:               "login button",
			actionType:         "click",
			elementLabel:       "Sign In",
			elementType:        "button",
			wantReversibility:  ReversibilityWriteReversible,
		},
		{
			name:               "delete action",
			actionType:         "click",
			elementLabel:       "Delete Account",
			elementType:        "button",
			wantReversibility:  ReversibilityWriteIrreversible,
		},
		{
			name:               "post tweet",
			actionType:         "click",
			elementLabel:       "Post Tweet",
			elementType:        "button",
			wantReversibility:  ReversibilityWriteIrreversible,
		},
		{
			name:               "password field",
			actionType:         "fill",
			elementLabel:       "Password",
			elementType:        "password",
			wantReversibility:  ReversibilitySensitiveWrite,
		},
		{
			name:               "search box",
			actionType:         "fill",
			elementLabel:       "Search",
			elementType:        "textbox",
			wantReversibility:  ReversibilityWriteReversible,
		},
		{
			name:               "navigate",
			actionType:         "navigate",
			elementLabel:       "",
			elementType:        "",
			wantReversibility:  ReversibilityRead,
		},
		{
			name:               "snapshot",
			actionType:         "snapshot",
			elementLabel:       "",
			elementType:        "",
			wantReversibility:  ReversibilityRead,
		},
		{
			name:               "remove item",
			actionType:         "click",
			elementLabel:       "Remove",
			elementType:        "button",
			wantReversibility:  ReversibilityWriteIrreversible,
		},
		{
			name:               "submit form",
			actionType:         "click",
			elementLabel:       "Submit",
			elementType:        "button",
			wantReversibility:  ReversibilityWriteIrreversible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyAction(tt.actionType, tt.elementLabel, tt.elementType)
			if got != tt.wantReversibility {
				t.Errorf("ClassifyAction() = %v, want %v", got, tt.wantReversibility)
			}
		})
	}
}

func TestActionClassifier_RequiresConfirmation(t *testing.T) {
	classifier := NewActionClassifier()

	tests := []struct {
		name        string
		reversibility string
		want        bool
	}{
		{
			name:           "read - no confirmation",
			reversibility: ReversibilityRead,
			want:          false,
		},
		{
			name:           "write reversible - no confirmation",
			reversibility: ReversibilityWriteReversible,
			want:          false,
		},
		{
			name:           "write irreversible - needs confirmation",
			reversibility: ReversibilityWriteIrreversible,
			want:          true,
		},
		{
			name:           "sensitive write - needs confirmation",
			reversibility: ReversibilitySensitiveWrite,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.RequiresConfirmation(tt.reversibility)
			if got != tt.want {
				t.Errorf("RequiresConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionClassifier_IsSafe(t *testing.T) {
	classifier := NewActionClassifier()

	tests := []struct {
		name        string
		reversibility string
		want        bool
	}{
		{
			name:           "read - safe",
			reversibility: ReversibilityRead,
			want:          true,
		},
		{
			name:           "write reversible - safe",
			reversibility: ReversibilityWriteReversible,
			want:          true,
		},
		{
			name:           "write irreversible - not safe",
			reversibility: ReversibilityWriteIrreversible,
			want:          false,
		},
		{
			name:           "sensitive write - not safe",
			reversibility: ReversibilitySensitiveWrite,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.IsSafe(tt.reversibility)
			if got != tt.want {
				t.Errorf("IsSafe() = %v, want %v", got, tt.want)
			}
		})
	}
}
