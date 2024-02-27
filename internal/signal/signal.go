// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

var (
	shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

// Callback is a callback that will be invoked when one of the shutdownSignals
// is caught. It is required that the implementation of this function should be very quick to ensure that it
// finishes in time before the subsequent signal is caught which will result in a forced exit of the app.
type Callback[T any] func(os.Signal, T) error

// SetupHandler sets up a context which reacts to shutdownSignals
func SetupHandler[T any](logger *zap.Logger, callback Callback[T], callbackParam T) (context.Context, context.CancelFunc) {
	ctx, cancelFn := context.WithCancel(context.Background())
	notifierCh := make(chan os.Signal, 1)
	signal.Notify(notifierCh, shutdownSignals...)

	go func() {
		sig := <-notifierCh
		//invoking the callback to process the signal
		if err := callback(sig, callbackParam); err != nil {
			logger.Error("failed to capture exit code", zap.Error(err))
		}
		logger.Info("caught shutdown signal", zap.Any("signal", sig))
		cancelFn()
		<-notifierCh
		os.Exit(1)
	}()
	return ctx, cancelFn
}
