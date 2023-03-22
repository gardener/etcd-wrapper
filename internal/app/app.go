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

package app

import (
	"context"

	"github.com/gardener/etcd-wrapper/internal/bootstrap"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

// Application is a top level struct which serves as an entry point for this application.
type Application struct {
	ctx             context.Context
	etcdInitializer bootstrap.EtcdInitializer
	cfg             *embed.Config
	etcdClient      *clientv3.Client
	logger          *zap.Logger
}

func NewApplication(ctx context.Context, sidecarConfig *bootstrap.SidecarConfig, logger *zap.Logger) (*Application, error) {
	etcdInitializer, err := bootstrap.NewEtcdInitializer(sidecarConfig, logger)
	if err != nil {
		return nil, err
	}
	return &Application{
		ctx:             ctx,
		etcdInitializer: etcdInitializer,
		logger:          logger,
	}, nil
}

// Setup sets up the application.
func (a *Application) Setup() error {
	// Set up etcd
	cfg, err := a.etcdInitializer.Run(a.ctx)
	if err != nil {
		return err
	}
	a.cfg = cfg
	return nil
}

// Start starts this application.
func (a *Application) Start() error {
	// Create etcd client for readiness probe
	cli, err := a.createEtcdClient()
	if err != nil {
		return err
	}
	a.etcdClient = cli
	defer func() {
		_ = a.etcdClient.Close()
	}()

	// Setup readiness probe
	go a.SetupReadinessProbe()

	// Create embedded etcd and start.
	_, err = embed.StartEtcd(a.cfg)
	if err != nil {
		return err
	}
	// Delete validation marker after etcd starts successfully
	_ = bootstrap.CleanupExitCode(bootstrap.DefaultExitCodeFilePath)
	<-a.ctx.Done()
	return nil
}
