package app

import (
	"context"

	"go.uber.org/zap"
)

// Application is a top level struct which serves as an entry point for this application.
type Application struct {
	ctx        context.Context
	logger     *zap.Logger
	caCertPath string
}

// Setup sets up the application.
func (a *Application) Setup() error {
	//sets up etcd by calling bootstrap.InitializeEtcd()
	return nil
}

// Start starts this application.
func (a *Application) Start() error {
	go a.SetupReadinessProbe()
	// Create embedded etcd and start.
	return nil
}
