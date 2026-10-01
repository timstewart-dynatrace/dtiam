// Package main is the entry point for the dtiam CLI.
package main

import (
	"github.com/jtimothystewart/dtiam/internal/cli"
	"github.com/jtimothystewart/dtiam/internal/commands/account"
	"github.com/jtimothystewart/dtiam/internal/commands/analyze"
	applycmd "github.com/jtimothystewart/dtiam/internal/commands/apply"
	"github.com/jtimothystewart/dtiam/internal/commands/boundary"
	"github.com/jtimothystewart/dtiam/internal/commands/bulk"
	"github.com/jtimothystewart/dtiam/internal/commands/cache"
	configcmd "github.com/jtimothystewart/dtiam/internal/commands/config"
	"github.com/jtimothystewart/dtiam/internal/commands/create"
	deletecmd "github.com/jtimothystewart/dtiam/internal/commands/delete"
	"github.com/jtimothystewart/dtiam/internal/commands/describe"
	"github.com/jtimothystewart/dtiam/internal/commands/doctor"
	"github.com/jtimothystewart/dtiam/internal/commands/export"
	"github.com/jtimothystewart/dtiam/internal/commands/get"
	"github.com/jtimothystewart/dtiam/internal/commands/group"
	"github.com/jtimothystewart/dtiam/internal/commands/serviceuser"
	templatecmd "github.com/jtimothystewart/dtiam/internal/commands/template"
	"github.com/jtimothystewart/dtiam/internal/commands/user"
)

func main() {
	// Register commands
	cli.AddCommand(configcmd.Cmd)
	cli.AddCommand(get.Cmd)
	cli.AddCommand(describe.Cmd)
	cli.AddCommand(create.Cmd)
	cli.AddCommand(deletecmd.Cmd)
	cli.AddCommand(user.Cmd)
	cli.AddCommand(serviceuser.Cmd)
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

	// Execute
	cli.Execute()
}
