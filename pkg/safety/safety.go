// Package safety decides whether an operation is allowed under a context's
// safety level.
//
// Commands declare what they do (see Operation); the root command checks that
// declaration against the active context before the command runs. Keeping the
// decision here, rather than in each command, means a new command cannot
// forget to check -- it can only forget to declare, which the command-tree
// test catches.
package safety

import (
	"fmt"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
)

// Operation is the kind of change a command makes.
type Operation string

const (
	// Read changes nothing on the account.
	Read Operation = "read"
	// Create adds resources or grants access (new memberships, bindings).
	Create Operation = "create"
	// Update changes existing resources in place.
	Update Operation = "update"
	// Delete removes resources or removes access: deleting objects, removing
	// members, detaching boundaries, revoking permissions, replacing a set.
	Delete Operation = "delete"
)

// AnnotationKey is the cobra annotation a command uses to declare its Operation.
const AnnotationKey = "dtiam.io/operation"

// ParseOperation validates an operation name.
func ParseOperation(s string) (Operation, error) {
	switch Operation(s) {
	case Read, Create, Update, Delete:
		return Operation(s), nil
	}
	return "", fmt.Errorf("unknown operation %q", s)
}

// BlockedError reports an operation the context's safety level forbids.
type BlockedError struct {
	Context   string
	Level     string
	Operation Operation
	Command   string
}

func (e *BlockedError) Error() string {
	ctx := e.Context
	if ctx == "" {
		ctx = "(no context)"
	}
	return fmt.Sprintf("%q is a %s operation, which context %q (safety level %s) does not allow; "+
		"use another context, or change it with 'dtiam config set-context %s --safety-level LEVEL'",
		e.Command, e.Operation, ctx, e.Level, ctx)
}

// Allowed reports whether op is permitted at level.
func Allowed(level string, op Operation) bool {
	switch level {
	case config.SafetyReadOnly:
		return op == Read
	case config.SafetyNoDelete:
		return op != Delete
	case config.SafetyReadWrite:
		return true
	default:
		// Unknown levels fail closed.
		return op == Read
	}
}

// Check returns a *BlockedError when op is not permitted at level.
func Check(contextName, level string, op Operation, command string) error {
	if Allowed(level, op) {
		return nil
	}
	return &BlockedError{Context: contextName, Level: level, Operation: op, Command: command}
}
