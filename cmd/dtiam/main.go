// Package main is the entry point for the dtiam CLI.
package main

import (
	"github.com/timstewart-dynatrace/dtiam/v3/internal/cli"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/account"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/analyze"
	applycmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/apply"
	authcmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/auth"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/boundary"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/bulk"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/cache"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/catalog"
	configcmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/config"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/create"
	deletecmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/delete"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/describe"
	diffcmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/diff"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/doctor"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/export"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/get"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/group"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/serviceuser"
	templatecmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/template"
	tokencmd "github.com/timstewart-dynatrace/dtiam/v3/internal/commands/token"
	"github.com/timstewart-dynatrace/dtiam/v3/internal/commands/user"
)

func main() {
	registerCommands()
	cli.Execute()
}

// registerCommands builds the command tree. It is separate from main so tests
// can inspect the complete tree.
func registerCommands() {
	cli.AddCommand(configcmd.Cmd)
	cli.AddCommand(get.Cmd)
	cli.AddCommand(describe.Cmd)
	cli.AddCommand(create.Cmd)
	cli.AddCommand(deletecmd.Cmd)
	cli.AddCommand(user.Cmd)
	cli.AddCommand(serviceuser.Cmd)
	cli.AddCommand(tokencmd.Cmd)
	cli.AddCommand(group.Cmd)
	cli.AddCommand(boundary.Cmd)
	cli.AddCommand(account.Cmd)
	cli.AddCommand(cache.Cmd)
	cli.AddCommand(bulk.Cmd)
	cli.AddCommand(export.Cmd)
	cli.AddCommand(analyze.Cmd)
	cli.AddCommand(templatecmd.Cmd)
	cli.AddCommand(applycmd.Cmd)
	cli.AddCommand(doctor.Cmd)
	cli.AddCommand(diffcmd.Cmd)
	cli.AddCommand(authcmd.Cmd)
	cli.AddCommand(catalog.Cmd)

	annotateOperations(cli.RootCmd)
	cli.RejectUnknownSubcommands(cli.RootCmd)
}
