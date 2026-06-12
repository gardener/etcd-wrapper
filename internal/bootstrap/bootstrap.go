// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	"github.com/gardener/etcd-wrapper/internal/util"

	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	sigsyaml "sigs.k8s.io/yaml"
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
func NewEtcdInitializer(brConfig *types.BackupRestoreConfig, logger *zap.Logger) (EtcdInitializer, error) {
	// Validate backup-restore configuration
	if err := brConfig.Validate(); err != nil {
		return nil, err
	}

	//create backup-restore client
	brClient, err := brclient.NewDefaultClient(*brConfig)
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
			i.logger.Info("Fetched initialization status is `New`. Triggering etcd initialization with validation mode", zap.Any("mode", validationMode))
			if err = i.brClient.TriggerInitialization(ctx, validationMode); err != nil {
				i.logger.Error("error while triggering initialization to backup-restore", zap.Error(err))
			}
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

// ChangeFilePermissions changes the file permissions of all files in the given directory and its subdirectories recursively.
func ChangeFilePermissions(dir string, mode os.FileMode) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error stating directory %s: %w", dir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", dir)
	}

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error walking the path %q: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		if err = os.Chmod(path, mode); err != nil {
			return fmt.Errorf("error changing file permissions for %q: %w", path, err)
		}
		return nil
	})
}

// CaptureExitCode captures the exit signal into a file `exit_code`
func CaptureExitCode(signal os.Signal, exitCodeFilePath string) error {
	if signal == nil {
		return nil
	}
	interruptSignal := []byte(signal.String())
	return os.WriteFile(exitCodeFilePath, interruptSignal, 0600)
}

// CleanupExitCode removes the `exit_code` file
func CleanupExitCode(exitCodeFilePath string) error {
	err := os.Remove(exitCodeFilePath)
	if errors.Is(err, os.ErrNotExist) {
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
	cfg, err := embed.ConfigFromFile(etcdConfigFilePath)
	if err != nil {
		return nil, err
	}
	if err := applyPeerSkipClientSANVerify(etcdConfigFilePath, cfg, i.logger); err != nil {
		return nil, fmt.Errorf("failed to apply peer-transport-security.skip-client-san-verification: %w", err)
	}
	return cfg, nil
}

// etcdConfigWithPeerSkipSAN parses only the nested "peer-transport-security.skip-client-san-verification"
// key present in the etcd config-map template passed by etcd-druid.
type etcdConfigWithPeerSkipSAN struct {
	PeerTransportSecurity struct {
		SkipClientSANVerification bool `json:"skip-client-san-verification"`
	} `json:"peer-transport-security"`
}

// applyPeerSkipClientSANVerify reads the etcd config file at configFilePath and
// sets cfg.PeerTLSInfo.SkipClientSANVerify to the value of the nested
// peer-transport-security.skip-client-san-verification key. An absent key is
// treated as false.
//
// TODO(@seshachalam-yv): remove this handling once the project moves to etcd
// v3.6.x — etcd v3.6 honors peer-transport-security.skip-client-san-verification
// natively in its config file, so embed.ConfigFromFile will populate
// cfg.PeerTLSInfo.SkipClientSANVerify and this shim is no longer needed.
func applyPeerSkipClientSANVerify(configFilePath string, cfg *embed.Config, logger *zap.Logger) error {
	data, err := os.ReadFile(configFilePath) // #nosec G304 -- configFilePath is from trusted backup-restore service
	if err != nil {
		return fmt.Errorf("failed to read etcd config file: %w", err)
	}
	parsed := etcdConfigWithPeerSkipSAN{}
	if err := sigsyaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("failed to parse etcd config: %w", err)
	}
	cfg.PeerTLSInfo.SkipClientSANVerify = parsed.PeerTransportSecurity.SkipClientSANVerification
	if cfg.PeerTLSInfo.SkipClientSANVerify {
		logger.Info("Set PeerTLSInfo.SkipClientSANVerify=true from peer-transport-security.skip-client-san-verification")
	}
	return nil
}

func determineValidationMode(exitCodeFilePath string, logger *zap.Logger) brclient.ValidationType {
	var err error

	// remove legacy validation_marker file created by etcd-custom-image
	if err = CleanupExitCode(types.ValidationMarkerFilePath); err != nil {
		logger.Error("error in removing validation_marker file", zap.String("validationMarkerFilePath", types.ValidationMarkerFilePath), zap.Error(err))
	}

	if _, err = os.Stat(exitCodeFilePath); err == nil {
		data, err := os.ReadFile(exitCodeFilePath) // #nosec G304 -- only path passed is `DefaultExitCodeFilePath`, no user input is used.
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
