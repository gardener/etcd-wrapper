// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLoggerConfig configures a default Zap logger.
func SetupLoggerConfig(level zapcore.Level) *zap.Config {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderConfig
	cfg.Level = zap.NewAtomicLevelAt(level)
	return &cfg
}
