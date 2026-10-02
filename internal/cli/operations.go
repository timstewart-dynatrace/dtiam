package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/pkg/config"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
)

// escalationPrefix prefixes annotations that raise a command's operation when
// a flag is set: "dtiam.io/escalate/<flag>" = "<operation>".
const escalationPrefix = "dtiam.io/escalate/"

// SetOperation declares what a command does to the account.
func SetOperation(c *cobra.Command, op safety.Operation) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[safety.AnnotationKey] = string(op)
}

// SetFlagEscalation declares that setting flag makes the command an op.
func SetFlagEscalation(c *cobra.Command, flag string, op safety.Operation) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[escalationPrefix+flag] = string(op)
}

// RelativePath returns a command's path below the root ("group update").
func RelativePath(c *cobra.Command) string {
	path := c.CommandPath()
	if root := c.Root(); root != nil {
		path = strings.TrimPrefix(strings.TrimPrefix(path, root.Name()), " ")
	}
	return path
}

// DeclaredOperation returns the operation a command declares, inherited from
// its nearest annotated ancestor. ok is false when nothing in the chain
// declares one.
func DeclaredOperation(c *cobra.Command) (op safety.Operation, ok bool) {
	for cur := c; cur != nil; cur = cur.Parent() {
		if v, found := cur.Annotations[safety.AnnotationKey]; found {
			parsed, err := safety.ParseOperation(v)
			if err != nil {
				return "", false
			}
			return parsed, true
		}
	}
	return "", false
}

// EffectiveOperation is the declared operation, raised by any escalation
// whose flag is set on this invocation.
func EffectiveOperation(c *cobra.Command) (safety.Operation, bool) {
	op, ok := DeclaredOperation(c)
	if !ok {
		return "", false
	}
	for key, v := range c.Annotations {
		flag, isEscalation := strings.CutPrefix(key, escalationPrefix)
		if !isEscalation || !c.Flags().Changed(flag) {
			continue
		}
		if set, _ := c.Flags().GetBool(flag); !set {
			continue
		}
		if escalated, err := safety.ParseOperation(v); err == nil {
			op = escalated
		}
	}
	return op, true
}

// checkSafety enforces the active context's safety level for a command.
//
// Commands that declare no operation are treated as mutating and checked as
// Delete. The command-tree test makes that unreachable in practice; failing
// closed is the backstop. Dry runs are allowed at every level, since they
// change nothing -- previewing a change is exactly what a read-only context is
// for.
func checkSafety(c *cobra.Command) error {
	op, ok := EffectiveOperation(c)
	if !ok {
		op = safety.Delete
	}
	if op == safety.Read || GlobalState.IsDryRun() {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		// No config yet: environment-variable credentials, no context, no
		// safety level to enforce.
		return nil
	}
	ctx := cfg.GetCurrentContext()
	if ctx == nil {
		return nil
	}
	return safety.Check(cfg.CurrentContext, ctx.EffectiveSafetyLevel(), op, c.CommandPath())
}

// RejectUnknownSubcommands makes every command that only groups subcommands
// fail on an unknown subcommand.
//
// By default cobra answers "dtiam get groupz" by printing get's help and
// exiting 0, without running any hooks: a typo looks like success to a script,
// and agent mode never starts. With NoArgs, cobra reports
// `unknown command "groupz" for "dtiam get"` instead; a bare "dtiam get" still
// shows the help.
func RejectUnknownSubcommands(root *cobra.Command) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c != root && c.HasSubCommands() && c.Run == nil && c.RunE == nil {
			c.Args = cobra.NoArgs
			c.RunE = func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}
