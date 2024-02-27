// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/cmd"
	"github.com/gardener/etcd-wrapper/internal/bootstrap"
	"github.com/gardener/etcd-wrapper/internal/signal"
	"go.uber.org/zap"
)

func main() {
	args := os.Args[1:]
	checkArgs(args)

	//create logger
	loggerCfg := bootstrap.SetupLoggerConfig(types.DefaultLogLevel)
	logger, err := loggerCfg.Build()
	if err != nil {
		log.Fatalf("error creating zap logger %v", err)
	}

	//setup signal handler
	ctx, cancelFn := signal.SetupHandler(logger, bootstrap.CaptureExitCode, types.DefaultExitCodeFilePath)

	// Add flags
	fs := flag.CommandLine
	cmd.EtcdCmd.AddFlags(fs)
	if err = fs.Parse(args[1:]); err != nil {
		logger.Fatal("error parsing command flags", zap.Error(err))
	}

	// Print all flags
	printFlags(logger)

	// InitAndStartEtcd command
	if err = cmd.EtcdCmd.Run(ctx, cancelFn, logger); err != nil {
		logger.Fatal("error during start or run of etcd", zap.Error(err))
	}
}

// checkArgs checks the command arguments and prints the usage if either the command name itself is not specified
// or the command specified is not supported.
func checkArgs(args []string) {
	//check if any unsupported command is specified. Print help if that is the case
	if len(args) < 1 || !cmd.IsCommandSupported(args[0]) {
		_ = cmd.PrintHelp(os.Stderr)
		os.Exit(1)
	}
}

func printFlags(logger *zap.Logger) {
	var flagsToPrint string
	flag.VisitAll(func(f *flag.Flag) {
		flagsToPrint += fmt.Sprintf("%s: %s, ", f.Name, f.Value)
	})
	logger.Info(fmt.Sprintf("Running with flags: %s", strings.TrimSuffix(flagsToPrint, ", ")))
}
