// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
