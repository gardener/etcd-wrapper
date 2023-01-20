package app

import (
	"context"

	"go.uber.org/zap"
)

type Application struct {
	ctx        context.Context
	logger     *zap.Logger
	caCertPath string
}

func (a *Application) Setup() error {
	//sets up etcd by calling bootstrap.InitializeEtcd()
	return nil
}

func (a *Application) Start() error {
	go a.SetupReadinessProbe()
	// Create embedded etcd and start.
	return nil
}
