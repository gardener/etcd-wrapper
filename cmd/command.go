package cmd

import (
	"context"
	"flag"

	"go.uber.org/zap"
)

// Command is a template for all commands.
type Command struct {
	// Name is the name of the command.
	Name string
	// ShortDesc is the short description of the command.
	ShortDesc string
	// UsageLine is the text describing the usage of the command.
	UsageLine string
	// LongDesc is the text containing the details of the command.
	LongDesc string
	// AddFlags provides a generic way for commands to initialize flags to the passed in FlagSet.
	AddFlags func(set *flag.FlagSet)
	// Run invokes the command.
	Run func(context.Context, context.CancelFunc, *zap.Logger) error
}

var (
	// Commands is a list of possible commands that could be run
	Commands = []*Command{
		&EtcdCmd,
	}
)

// IsCommandSupported checks if the command with the passed in commandName is a supported command.
func IsCommandSupported(commandName string) bool {
	for _, cmd := range Commands {
		if cmd.Name == commandName {
			return true
		}
	}
	return false
}
