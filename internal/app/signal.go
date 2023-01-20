package app

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

// SignalCallback is a callback that will be invoked when one of the shutdownSignals
// is caught. It is required that the implementation of this function should be very quick to ensure that it
// finishes in time before the subsequent signal is caught which will result in a forced exit of the app.
type SignalCallback func(os.Signal)

// SetupSignalHandler sets up a context which reacts to shutdownSignals
func SetupSignalHandler(logger *zap.Logger, callback SignalCallback) context.Context {
	ctx, cancelFn := context.WithCancel(context.Background())
	notifierCh := make(chan os.Signal, 1)
	signal.Notify(notifierCh, shutdownSignals...)

	go func() {
		sig := <-notifierCh
		logger.Info("caught shutdown signal", zap.Any("signal", sig))
		cancelFn()
		callback(sig) //invoking the callback to process the signal
		<-notifierCh
		os.Exit(1)
	}()
	return ctx
}
