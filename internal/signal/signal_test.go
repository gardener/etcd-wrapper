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
	callbackFn := func(signal os.Signal, _ string) {
		if signal != nil {
			fmt.Println("callback called.")
			receivedSignal = signal.String()
		}
	}

	ctx, _ := SetupHandler(logger, callbackFn, "")
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	g.Expect(err).To(BeNil())
	time.Sleep(5 * time.Second)
	g.Expect(ctx.Err()).To(Equal(context.Canceled))
	g.Expect(receivedSignal).To(Equal(os.Interrupt.String()))
}
