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
type Callback[T any] func(os.Signal, T)

// SetupHandler sets up a context which reacts to shutdownSignals
func SetupHandler[T any](logger *zap.Logger, callback Callback[T], callbackParam T) context.Context {
	ctx, cancelFn := context.WithCancel(context.Background())
	notifierCh := make(chan os.Signal, 1)
	signal.Notify(notifierCh, shutdownSignals...)

	go func() {
		sig := <-notifierCh
		callback(sig, callbackParam) //invoking the callback to process the signal
		logger.Info("caught shutdown signal", zap.Any("signal", sig))
		cancelFn()
		<-notifierCh
		os.Exit(1)
	}()
	return ctx
}
