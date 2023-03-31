package util

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/zap/zaptest"
)

var (
	retryResults     []string
	errAttemptFailed = errors.New("attempt failed")
)

const (
	AttemptSuccessful = "attempt-successful"
	AttemptFailed     = "attempt-failed"
)

func TestErrorIfExceedsAttempts(t *testing.T) {
	const (
		numAttempts = 5
		backOff     = 1 * time.Second
		operation   = "always-fails"
	)

	table := []struct {
		description            string
		canRetryFn             CanRetryPredicate
		expectedRetryResultLen int
	}{
		{"num attempts exhausted", alwaysRetry, numAttempts},
		{"neverRetry short-circuits num-attempts", neverRetry, 1},
	}

	for _, entry := range table {
		g := NewWithT(t)
		logger := zaptest.NewLogger(t)
		t.Run(entry.description, func(t *testing.T) {
			defer clearRetryResults()
			result := Retry(context.Background(), logger, operation, neverSucceeds, numAttempts, backOff, entry.canRetryFn)
			g.Expect(result.Value).To(Equal(AttemptFailed))
			g.Expect(result.Err).To(Equal(errAttemptFailed))
			g.Expect(len(retryResults)).To(Equal(entry.expectedRetryResultLen))
		})
	}
}

func TestSuccessWhenEventuallySucceeds(t *testing.T) {
	g := NewWithT(t)
	logger := zaptest.NewLogger(t)
	defer clearRetryResults()
	const (
		numAttempts      = 5
		backOff          = 1 * time.Second
		operation        = "eventually-succeeds"
		succeedAtAttempt = 3
	)
	var retryCount int
	retryFn := func() (string, error) {
		retryCount++
		if retryCount == succeedAtAttempt {
			retryResults = append(retryResults, AttemptSuccessful)
			return AttemptSuccessful, nil
		}
		retryResults = append(retryResults, AttemptFailed)
		return AttemptFailed, errAttemptFailed
	}

	result := Retry(context.Background(), logger, operation, retryFn, numAttempts, backOff, alwaysRetry)
	g.Expect(result.Value).To(Equal(AttemptSuccessful))
	g.Expect(result.Err).To(BeNil())
	g.Expect(len(retryResults)).To(Equal(succeedAtAttempt))
	g.Expect(retryResults).Should(ConsistOf(AttemptFailed, AttemptFailed, AttemptSuccessful))
}

func TestRetryWhenContextCancelled(t *testing.T) {
	g := NewWithT(t)
	logger := zaptest.NewLogger(t)
	defer clearRetryResults()
	const (
		numAttempts               = 5
		backOff                   = 1 * time.Second
		operation                 = "context-cancelled"
		contextCancelledAtAttempt = 2
	)
	ctx, cancelFn := context.WithCancel(context.Background())
	var retryCount int
	retryFn := func() (string, error) {
		retryCount++
		if retryCount == contextCancelledAtAttempt {
			cancelFn()
		} else {
			retryResults = append(retryResults, AttemptFailed)
		}
		return AttemptFailed, errAttemptFailed
	}

	result := Retry(ctx, logger, operation, retryFn, numAttempts, backOff, alwaysRetry)
	g.Expect(result.Value).To(BeEmpty())
	g.Expect(result.Err).To(Equal(context.Canceled))
	g.Expect(len(retryResults)).To(Equal(1))
}

func neverSucceeds() (string, error) {
	retryResults = append(retryResults, AttemptFailed)
	return AttemptFailed, errAttemptFailed
}

func clearRetryResults() {
	retryResults = nil
}

func alwaysRetry(_ error) bool {
	return true
}

func neverRetry(_ error) bool {
	return false
}
