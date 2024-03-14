// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CanRetryPredicate provides a predicate whose return value decides if an error can be ignored for the next retry.
type CanRetryPredicate func(err error) bool

// RetriableFunc is any function that should be retried. It returns any T and an error.
type RetriableFunc[T any] func() (T, error)

// Result encapsulates the result of retrying a function via Retry.
type Result[T any] struct {
	Value T
	Err   error
}

// IsErr checks if the Result has error set. If error is not nil then it will return true else it will return false.
func (r Result[T]) IsErr() bool {
	return r.Err != nil
}

// Retry will retry invoking the given function `fn` a max of `numAttempts` with a back off between successive attempts. If an invocation of the function returns an error
// then it will check it can proceed with the retry by evaluating via `canRetryFn`. A Result containing the return value if function invocation was successful or an error
// if the function was not successful will be returned to the caller. The caller can check if the Result is an error by invoking Result.IsErr function.
func Retry[T any](ctx context.Context, logger *zap.Logger, operation string, fn RetriableFunc[T], numAttempts int, backOff time.Duration, canRetryFn CanRetryPredicate) Result[T] {
	var (
		resultVal T
		err       error
	)
	for i := 0; i < numAttempts; i++ {
		select {
		case <-ctx.Done():
			logger.Error("context has been cancelled. stopping retry", zap.String("operation", operation), zap.Error(ctx.Err()))
			return Result[T]{Err: ctx.Err()}
		default:
		}
		// invoke the retriable function
		resultVal, err = fn()
		if err == nil {
			return Result[T]{Value: resultVal}
		}
		if !canRetryFn(err) {
			return Result[T]{Value: resultVal, Err: err}
		}
		select {
		case <-ctx.Done():
			logger.Error("context has been cancelled. stopping retry", zap.String("operation", operation), zap.Error(ctx.Err()))
			return Result[T]{Err: ctx.Err()}
		case <-time.After(backOff):
			logger.Info("re-attempting operation", zap.String("operation", operation), zap.Int("current-attempt", i), zap.Error(err))
		}
	}
	logger.Error("all retries exhausted", zap.String("operation", operation), zap.Int("numAttempts", numAttempts))
	return Result[T]{Value: resultVal, Err: err}
}

// AlwaysRetry ignores the error and always returns true. This can be used as a CanRetryPredicate when using Retry
// if a retry is required irrespective of any error during the invocation of the function.
func AlwaysRetry(_ error) bool {
	return true
}
