package main

import (
	"github.com/spf13/cobra"

	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/pkg/safety"
)

// operations declares what each command does to the account, by command path
// below the root. A subcommand inherits its nearest ancestor's operation, so a
// verb such as "delete" covers all of its resources; entries for a subcommand
// override the inherited value.
//
// Classification rule: anything that removes an object or removes access is
// Delete -- detaching a boundary widens what a binding allows, and replacing a
// group set drops memberships -- so the no-delete level blocks all of it.
// Local-only commands (config, template save/delete, cache) are Read: they never
// touch the account.
//
// TestEveryCommandDeclaresAnOperation fails for any command that resolves to
// nothing, so a new command cannot slip past the safety check by omission.
var operations = map[string]safety.Operation{
	// Read-only verbs.
	"get":      safety.Read,
	"describe": safety.Read,
	"export":   safety.Read,
	"analyze":  safety.Read,
	"account":  safety.Read,
	"diff":     safety.Read,
	"doctor":   safety.Read,
	"version":  safety.Read,
	"cache":    safety.Read,
	"config":   safety.Read,

	// Mutating verbs.
	"create": safety.Create,
	"delete": safety.Delete,
	"apply":  safety.Update, // creates or updates; never deletes

	// Templates: only "apply" reaches the account.
	"template":       safety.Read,
	"template apply": safety.Create,

	// User membership.
	"user":                    safety.Read,
	"user create":             safety.Create,
	"user add-to-groups":      safety.Create,
	"user remove-from-groups": safety.Delete,
	"user replace-groups":     safety.Delete,

	// Service users.
	"service-user":                   safety.Read,
	"service-user create":            safety.Create,
	"service-user update":            safety.Update,
	"service-user add-to-group":      safety.Create,
	"service-user remove-from-group": safety.Delete,
	"service-user delete":            safety.Delete,

	// Groups.
	"group":                   safety.Read,
	"group add-member":        safety.Create,
	"group remove-member":     safety.Delete,
	"group clone":             safety.Create,
	"group setup":             safety.Create,
	"group update":            safety.Update,
	"group grant-permission":  safety.Create, // Delete with --replace, see below
	"group revoke-permission": safety.Delete,

	// Boundaries.
	"boundary":                        safety.Read,
	"boundary attach":                 safety.Update,
	"boundary detach":                 safety.Delete,
	"boundary create-app-boundary":    safety.Create,
	"boundary create-schema-boundary": safety.Create,

	// Bulk.
	"bulk":                             safety.Create,
	"bulk export-group-members":        safety.Read,
	"bulk remove-users-from-group":     safety.Delete,
	"bulk create-groups-with-policies": safety.Create,

	// Platform tokens.
	"token":                safety.Update,
	"token set-expiration": safety.Update,

	// Identity and discovery.
	"auth":     safety.Read,
	"commands": safety.Read,
}

// escalations raise a command's operation when a flag is set, for commands
// whose effect depends on it.
var escalations = map[string]map[string]safety.Operation{
	// --replace overwrites the group's permission set, dropping the rest.
	"group grant-permission": {"replace": safety.Delete},
}

// annotateOperations writes the operations table onto the command tree.
func annotateOperations(root *cobra.Command) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		path := cli.RelativePath(c)
		if op, ok := operations[path]; ok {
			cli.SetOperation(c, op)
		}
		for flag, op := range escalations[path] {
			cli.SetFlagEscalation(c, flag, op)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}
