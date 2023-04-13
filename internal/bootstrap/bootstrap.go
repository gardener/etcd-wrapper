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

package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	"github.com/gardener/etcd-wrapper/internal/util"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

const (
	defaultBackupRestoreMaxRetries = 5
	defaultBackOffBetweenRetries   = 1 * time.Second
)

// EtcdInitializer is an interface for methods to be used to initialize etcd
type EtcdInitializer interface {
	Run(context.Context) (*embed.Config, error)
}

type initializer struct {
	brClient brclient.BackupRestoreClient
	logger   *zap.Logger
}

// NewEtcdInitializer creates and returns an EtcdInitializer object
func NewEtcdInitializer(sidecarConfig *types.SidecarConfig, logger *zap.Logger) (EtcdInitializer, error) {
	// Validate sidecar configuration
	if err := sidecarConfig.Validate(); err != nil {
		return nil, err
	}

	//create brclient
	brClient, err := brclient.NewDefaultClient(*sidecarConfig, brclient.DefaultEtcdConfigFilePath)
	if err != nil {
		return nil, err
	}

	return &initializer{
		brClient: brClient,
		logger:   logger,
	}, nil
}

// Run initializes the etcd and gets the etcd configuration
func (i *initializer) Run(ctx context.Context) (*embed.Config, error) {
	var (
		err        error
		initStatus brclient.InitStatus
	)
	for initStatus != brclient.Successful {
		if initStatus, err = i.brClient.GetInitializationStatus(ctx); err != nil {
			i.logger.Error("error while fetching initialization status", zap.Error(err))
		}
		i.logger.Info("Fetched initialization status", zap.String("Status", initStatus.String()))
		if initStatus == brclient.New {
			validationMode := determineValidationMode(types.DefaultExitCodeFilePath, i.logger)
			if err = i.brClient.TriggerInitialization(ctx, validationMode); err != nil {
				i.logger.Error("error while triggering initialization to backup-restore", zap.Error(err))
			}
			i.logger.Info("Fetched initialization status is `New`. Triggering etcd initialization with validation mode", zap.Any("mode", validationMode))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(defaultBackOffBetweenRetries):
		}
	}
	i.logger.Info("Etcd initialization succeeded")
	return i.tryGetEtcdConfig(ctx, defaultBackupRestoreMaxRetries, defaultBackOffBetweenRetries)
}

// CaptureExitCode captures the exit signal into a file `exit_code`
func CaptureExitCode(signal os.Signal, exitCodeFilePath string) {
	if signal != nil {
		interruptSignal := []byte(signal.String())
		_ = os.WriteFile(exitCodeFilePath, interruptSignal, 0644)
	}
}

// CleanupExitCode removes the `exit_code` file
func CleanupExitCode(exitCodeFilePath string) error {
	err := os.Remove(exitCodeFilePath)
	if errors.Is(err, os.ErrNotExist) {
		//log file does not exist
		return nil
	}
	return err
}

func (i *initializer) tryGetEtcdConfig(ctx context.Context, maxRetries int, interval time.Duration) (*embed.Config, error) {
	// Get etcd config only
	opResult := util.Retry[string](ctx, i.logger, "GetEtcdConfig", func() (string, error) {
		return i.brClient.GetEtcdConfig(ctx)
	}, maxRetries, interval, util.AlwaysRetry)
	if opResult.IsErr() {
		return nil, opResult.Err
	}
	etcdConfigFilePath := opResult.Value
	i.logger.Info("Fetched and written etcd configuration", zap.String("path", etcdConfigFilePath))
	return embed.ConfigFromFile(etcdConfigFilePath)
}

func determineValidationMode(exitCodeFilePath string, logger *zap.Logger) brclient.ValidationType {
	var err error
	if _, err = os.Stat(exitCodeFilePath); err == nil {
		data, err := os.ReadFile(exitCodeFilePath)
		if err != nil {
			logger.Error("error in reading exitCodeFile, assuming full-validation to be done.", zap.String("exitCodeFilePath", exitCodeFilePath), zap.Error(err))
			return brclient.FullValidation
		}
		validationMarker := strings.TrimSpace(string(data))
		if validationMarker == "terminated" || validationMarker == "interrupt" {
			logger.Info("last captured exit code read, assuming sanity validation to be done.", zap.String("exitCodeFilePath", exitCodeFilePath), zap.String("signal-captured", validationMarker))
			return brclient.SanityValidation
		}
	}
	logger.Error("error in checking if exitCodeFile exists, assuming full-validation to be done.", zap.String("exitCodeFilePath", exitCodeFilePath), zap.Error(err))
	// Full validation if error
	return brclient.FullValidation
}
