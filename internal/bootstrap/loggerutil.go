package bootstrap

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLogger configures a default Zap logger.
func SetupLogger(level zapcore.Level) (*zap.Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderConfig
	cfg.Level = zap.NewAtomicLevelAt(level)
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l, nil
}
