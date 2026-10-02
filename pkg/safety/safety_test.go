package safety

import (
	"errors"
	"testing"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
)

func TestAllowed(t *testing.T) {
	tests := []struct {
		level string
		op    Operation
		want  bool
	}{
		{config.SafetyReadOnly, Read, true},
		{config.SafetyReadOnly, Create, false},
		{config.SafetyReadOnly, Update, false},
		{config.SafetyReadOnly, Delete, false},
		{config.SafetyNoDelete, Read, true},
		{config.SafetyNoDelete, Create, true},
		{config.SafetyNoDelete, Update, true},
		{config.SafetyNoDelete, Delete, false},
		{config.SafetyReadWrite, Delete, true},
		{"bogus", Read, true},
		{"bogus", Create, false},
	}
	for _, tt := range tests {
		t.Run(tt.level+"/"+string(tt.op), func(t *testing.T) {
			if got := Allowed(tt.level, tt.op); got != tt.want {
				t.Errorf("Allowed(%q, %q) = %v, want %v", tt.level, tt.op, got, tt.want)
			}
		})
	}
}

func TestCheck_ReturnsBlockedError(t *testing.T) {
	err := Check("prod", config.SafetyReadOnly, Delete, "dtiam delete group")
	var blocked *BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("Check() = %v, want a *BlockedError", err)
	}
	if blocked.Context != "prod" || blocked.Operation != Delete {
		t.Errorf("BlockedError = %+v", blocked)
	}
	if Check("prod", config.SafetyReadWrite, Delete, "x") != nil {
		t.Error("readwrite should allow delete")
	}
}

func TestParseOperation(t *testing.T) {
	for _, s := range []string{"read", "create", "update", "delete"} {
		if _, err := ParseOperation(s); err != nil {
			t.Errorf("ParseOperation(%q) error: %v", s, err)
		}
	}
	if _, err := ParseOperation("write"); err == nil {
		t.Error("ParseOperation(write) should fail")
	}
}
