// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/internal/bootstrap"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/embed"
	"go.uber.org/zap"
)

// Application is a top level struct which serves as an entry point for this application.
type Application struct {
	ctx      context.Context
	cancelFn context.CancelFunc
	// Config is the application config
	Config           types.Config
	etcdInitializer  bootstrap.EtcdInitializer
	cfg              *embed.Config
	etcdClient       *clientv3.Client
	etcd             *embed.Etcd
	waitReadyTimeout time.Duration
	logger           *zap.Logger
	etcdReady        bool // should have only one actor that updates it, queryAndUpdateEtcdReadiness()
	server           *http.Server
}

// NewApplication initializes and returns an application struct
func NewApplication(ctx context.Context, cancelFn context.CancelFunc, config types.Config, waitReadyTimeout time.Duration, logger *zap.Logger) (*Application, error) {
	logger.Info("Initializing application", zap.Any("config", config))
	etcdInitializer, err := bootstrap.NewEtcdInitializer(&config.BackupRestore, logger)
	if err != nil {
		return nil, err
	}
	return &Application{
		ctx:              ctx,
		cancelFn:         cancelFn,
		Config:           config,
		etcdInitializer:  etcdInitializer,
		waitReadyTimeout: waitReadyTimeout,
		logger:           logger,
	}, nil
}

// Setup sets up etcd by triggering initialization of the etcd DB.
func (a *Application) Setup() error {
	// Set up etcd
	cfg, err := a.etcdInitializer.Run(a.ctx)
	if err != nil {
		return err
	}
	a.cfg = cfg

	syscall.Umask(0077)
	return nil
}

// Start sets up readiness probe and starts an embedded etcd.
func (a *Application) Start() error {
	var err error

	// Change file permissions for files previously created without umask 0077
	// TODO (shreyas-s-rao): remove this temporary code in etcd-wrapper v0.8.0
	if err = bootstrap.ChangeFilePermissions(a.cfg.Dir, 0640); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}

	// Create etcd client for readiness probe
	cli, err := a.createEtcdClient()
	if err != nil {
		return err
	}
	a.etcdClient = cli
	defer a.Close()

	// Setup readiness probe
	go a.queryAndUpdateEtcdReadiness()

	// start HTTP server to serve endpoints
	go a.startHTTPServer()
	defer func() {
		if err := a.stopHTTPServer(); err != nil {
			a.logger.Error("unable to stop HTTP server: %v",
				zap.Error(err),
			)
		}
	}()

	// Create embedded etcd and start.
	if err = a.startEtcd(); err != nil {
		return err
	}
	// Delete exit code file after etcd starts successfully
	if err = bootstrap.CleanupExitCode(types.DefaultExitCodeFilePath); err != nil {
		a.logger.Warn("failed to clean-up last captured exit code", zap.Error(err))
	}

	// block till application context is cancelled, or there is a notification on etcd.Server.StopNotify channel
	// or there is an error notification on etcd.Err channel
	select {
	case <-a.ctx.Done():
		a.logger.Error("application context has been cancelled", zap.Error(a.ctx.Err()))
	case <-a.etcd.Server.StopNotify():
		a.logger.Error("etcd server has been aborted, received notification on StopNotify channel")
	case err = <-a.etcd.Err():
		a.logger.Error("error received on etcd Err channel", zap.Error(err))
	}

	return nil
}

// Close closes resources(e.g. etcd client) and cancels the context if not already done so.
func (a *Application) Close() {
	if err := a.etcdClient.Close(); err != nil {
		a.logger.Error("failed to close etcd client", zap.Error(err))
	}
	if a.etcd != nil {
		a.etcd.Close()
	}
	a.cancelContext()
}

func (a *Application) cancelContext() {
	// only if the context has not yet been cancelled, call the context.CancelFunc
	if a.ctx.Err() == nil {
		a.cancelFn()
	}
}

func (a *Application) startEtcd() error {
	// TODO StartEtcd returns an Etcd object. In future we should use that to listen on leadership change notifications (when we move to a version of etcd which exposes the channel).
	etcd, err := embed.StartEtcd(a.cfg)
	if err != nil {
		return err
	}

	// wait till the etcd server notifies that it is ready, or if an abrupt stop has happened which is notified
	// via etcd.Server.Notify or there is a timeout waiting for the etcd server to start.
	select {
	case <-etcd.Server.ReadyNotify():
		a.logger.Info("etcd server is now ready to serve client requests")
	case <-etcd.Server.StopNotify():
		a.logger.Error("etcd server has been aborted, received notification on StopNotify channel")
	case <-time.After(a.waitReadyTimeout):
		a.logger.Error("timeout waiting for ReadyNotify signal, aborting start of etcd")
	}
	a.etcd = etcd
	return nil
}
