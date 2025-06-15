package resolve_dns

import (
	"context"
	"encoding/json"
	"os"

	"github.com/enternejp/tap-etherip-config-sync/internal/config"
	"github.com/enternejp/tap-etherip-config-sync/internal/dns"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func run(cmd *cobra.Command, args []string) error {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	loggerConfig.DisableStacktrace = true
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	l, err := loggerConfig.Build()
	if err != nil {
		return err
	}
	defer l.Sync()

	in, err := config.NewWithDNS(os.Stdin)
	if err != nil {
		return errors.Wrapf(err, "failed to parse input config")
	}

	out, err := resolveConfig(in, l)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func resolveConfig(in *config.ConfWithDNS, l *zap.Logger) (*config.Conf, error) {
	var out config.Conf
	ctx := context.Background()
	for _, tw := range in.Tunnels {
		var localIP, remoteIP string
		if tw.LocalFQDN != "" {
			ips, err := dns.ResolveAAAA(ctx, tw.LocalFQDN)
			if err != nil {
				l.Warn("LocalFQDN resolve failed", zap.String("fqdn", tw.LocalFQDN), zap.Error(err))
			} else {
				if len(ips) > 1 {
					l.Warn("LocalFQDN multiple AAAA records. Use first IPv6 address.", zap.String("fqdn", tw.LocalFQDN), zap.Strings("dns_answers", ips))
				}
				localIP = ips[0]
			}
		}
		if localIP == "" {
			localIP = tw.LocalIPAddr
		}
		if localIP == "" {
			l.Error("LocalFQDN and LocalIPAddr are both empty", zap.String("tunnel", tw.Name))
			continue
		}
		if tw.RemoteFQDN != "" {
			ips, err := dns.ResolveAAAA(ctx, tw.RemoteFQDN)
			if err != nil {
				l.Warn("RemoteFQDN resolve failed", zap.String("fqdn", tw.RemoteFQDN), zap.Error(err))
			} else {
				if len(ips) > 1 {
					l.Warn("RemoteFQDN multiple AAAA records. Use first IPv6 address.", zap.String("fqdn", tw.RemoteFQDN), zap.Strings("dns_answers", ips))
				}
				remoteIP = ips[0]
			}
		}
		if remoteIP == "" {
			remoteIP = tw.RemoteIPAddr
		}
		if remoteIP == "" {
			l.Error("RemoteFQDN and RemoteIPAddr are both empty", zap.String("tunnel", tw.Name))
			continue
		}
		out.Tunnels = append(out.Tunnels, config.Tunnel{
			Name:         tw.Name,
			LocalIPAddr:  localIP,
			RemoteIPAddr: remoteIP,
		})
	}
	return &out, nil
}
