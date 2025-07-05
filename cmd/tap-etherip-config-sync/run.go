package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/enternejp/tap-etherip-config-sync/internal/config"
	"github.com/enternejp/tap-etherip-config-sync/internal/tunnel"
)

func run(cmd *cobra.Command, _ []string) error {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	loggerConfig.DisableStacktrace = true

	logLevelStr := viper.GetString(flagLogLevel)
	logLevel, err := zap.ParseAtomicLevel(logLevelStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse log level: %s", logLevelStr)
	}
	loggerConfig.Level = zap.NewAtomicLevelAt(logLevel.Level())
	l, err := loggerConfig.Build()
	if err != nil {
		return err
	}
	defer l.Sync()

	configPath := viper.GetString(flagKeyConfig)
	f, err := os.Open(configPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open config file: %s", configPath)
	}
	defer f.Close()
	conf, err := config.New(f)
	if err != nil {
		return errors.Wrapf(err, "failed to parse config file: %s", configPath)
	}

	tunl := tunnel.Tunnel{EnvFileBasePath: viper.GetString(flagEnvBasePath)}
	tunnels, err := tunl.GetCurrentTunnels()
	if err != nil {
		return errors.Wrapf(err, "failed to get current tunnels")
	}
	l.Debug("current tunnels", zap.Any("tunnels", tunnels))

	diffs := tunnel.DiffTunnels(conf.Tunnels, tunnels)
	for _, d := range diffs {
		l.Debug("device diff", zap.String("name", d.Name), zap.String("action", d.Action.String()))
		switch d.Action {
		case tunnel.ActionCreate, tunnel.ActionRecreate:
			cfg := tunnel.TunnelConfig{
				Name:     d.Config.Name,
				LocalIP:  d.Config.LocalIPAddr,
				RemoteIP: d.Config.RemoteIPAddr,
			}
			if err := tunl.CreateOrRecreate(cfg); err != nil {
				l.Error("failed to create/recreate tunnel", zap.Error(err), zap.String("name", d.Name))
				continue
			}
			l.Info("tunnel created/recreated", zap.String("name", d.Name))
		case tunnel.ActionDelete:
			if err := tunl.Delete(d.Name); err != nil {
				l.Error("failed to delete tunnel", zap.Error(err), zap.String("name", d.Name))
				continue
			}
			l.Info("tunnel deleted", zap.String("name", d.Name))
		}
	}
	return nil
}
