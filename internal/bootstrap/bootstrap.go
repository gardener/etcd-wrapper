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
	"github.com/gardener/etcd-wrapper/internal/types"
	"os"
	"strings"
	"time"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	"github.com/gardener/etcd-wrapper/internal/util"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

const (
	defaultBackupRestoreMaxRetries = 5
	defaultBackOffBetweenRetries   = 1 * time.Second
)

type EtcdInitializer interface {
	Run(context.Context) (*embed.Config, error)
}

type initializer struct {
	brClient brclient.BackupRestoreClient
	logger   *zap.Logger
}

func NewEtcdInitializer(sidecarConfig *types.SidecarConfig, logger *zap.Logger) (EtcdInitializer, error) {
	// Validate sidecar configuration
	if err := sidecarConfig.Validate(); err != nil {
		return nil, err
	}
	//create brclient
	brClient, err := brclient.NewClient(*sidecarConfig, brclient.DefaultEtcdConfigFilePath)
	if err != nil {
		return nil, err
	}

	return &initializer{
		brClient: brClient,
		logger:   logger,
	}, nil
}

func (i *initializer) Run(ctx context.Context) (*embed.Config, error) {
	var (
		err        error
		initStatus brclient.InitStatus
	)
	for initStatus != brclient.Successful {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if initStatus, err = i.brClient.GetInitializationStatus(); err != nil {
			i.logger.Error("error while fetching initialization status", zap.Error(err))
		}
		i.logger.Info("Fetched initialization status", zap.String("Status", initStatus.String()))
		if initStatus == brclient.New {
			validationMode := getValidationMode(types.DefaultExitCodeFilePath)
			if err = i.brClient.TriggerInitialization(validationMode); err != nil {
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

func (i *initializer) tryGetEtcdConfig(ctx context.Context, maxRetries int, interval time.Duration) (*embed.Config, error) {
	// Get etcd config only
	opResult := util.Retry[string](ctx, i.logger, "GetEtcdConfig", func() (string, error) {
		return i.brClient.GetEtcdConfig()
	}, maxRetries, interval, util.AlwaysRetry)
	if opResult.IsErr() {
		return nil, opResult.Err
	}
	etcdConfigFilePath := opResult.Value
	i.logger.Info("Fetched etcd configuration")
	return embed.ConfigFromFile(etcdConfigFilePath)
}

func getValidationMode(exitCodeFilePath string) brclient.ValidationType {
	if _, err := os.Stat(exitCodeFilePath); err == nil {
		data, err := os.ReadFile(exitCodeFilePath)
		if err != nil {
			return brclient.FullValidation
		}
		validationMarker := strings.TrimSpace(string(data))
		if validationMarker == "terminated" || validationMarker == "interrupt" {
			return brclient.SanityValidation
		}
	}
	// Full validation if error
	return brclient.FullValidation
}

func CaptureExitCode(signal os.Signal, exitCodeFilePath string) {
	// capture the signal into exit_code
	// Write signal to validation marker
	interruptSignal := []byte(signal.String())
	_ = os.WriteFile(exitCodeFilePath, interruptSignal, 0644)
}

func CleanupExitCode(exitCodeFilePath string) error {
	//removes exit_code file
	err := os.Remove(exitCodeFilePath)
	if errors.Is(err, os.ErrNotExist) {
		//log file does not exist
		return nil
	}
	return err
}
