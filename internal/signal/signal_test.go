// Copyright 2023 SAP SE or an SAP affiliate company
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
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

func TestSetupSignalHandler(t *testing.T) {
	g := NewWithT(t)
	logger := zaptest.NewLogger(t)

	var (
		receivedSignal string
		err            error
	)
	callbackFn := func(signal os.Signal, _ string) error {
		if signal != nil {
			fmt.Println("callback called.")
			receivedSignal = signal.String()
		}
		return nil
	}

	ctx, _ := SetupHandler(logger, callbackFn, "")
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	g.Expect(err).To(BeNil())
	time.Sleep(5 * time.Second)
	g.Expect(ctx.Err()).To(Equal(context.Canceled))
	g.Expect(receivedSignal).To(Equal(os.Interrupt.String()))
}
