// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/cmd"
	"github.com/gardener/etcd-wrapper/internal/bootstrap"
	"github.com/gardener/etcd-wrapper/internal/signal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultLogLevel = zapcore.InfoLevel
)

func main() {
	args := os.Args[1:]
	checkArgs(args)
	logger, err := bootstrap.SetupLogger(defaultLogLevel)
	if err != nil {
		log.Printf("error creating zap logger %v", err)
		os.Exit(1)
	}

	ctx := signal.SetupHandler(logger, bootstrap.CaptureExitCode, types.DefaultExitCodeFilePath)

	// Add flags
	fs := flag.CommandLine
	cmd.EtcdCmd.AddFlags(fs)
	_ = fs.Parse(args[1:])

	// Print all flags
	printFlags(logger)

	// InitAndStartEtcd command
	if err = cmd.EtcdCmd.Run(ctx, logger); err != nil {
		logger.Fatal("error during start or run of etcd", zap.Error(err))
		os.Exit(1)
	}
}

// Should check if any arg is help and print
func checkArgs(args []string) {
	//check if any unsupported command is specified. Print help if that is the case
	if len(args) < 1 || !cmd.IsCommandSupported(args[0]) {
		cmd.PrintHelp(os.Stdout)
		os.Exit(0)
	}
}

func printFlags(logger *zap.Logger) {
	var flagsToPrint string
	flag.VisitAll(func(f *flag.Flag) {
		flagsToPrint += fmt.Sprintf("%s: %s, ", f.Name, f.Value)
	})

	logger.Info(fmt.Sprintf("Running with flags: %s", flagsToPrint[:len(flagsToPrint)-2]))
}
