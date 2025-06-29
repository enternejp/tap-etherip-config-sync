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
	for _, inTunl := range in.Tunnels {
		var localIP, remoteIP string

		if inTunl.LocalFQDN == "" && inTunl.LocalIPAddr == "" {
			l.Error("LocalFQDN and LocalIPAddr are both empty", zap.String("tunnel", inTunl.Name))
			continue
		}
		if inTunl.LocalFQDN != "" {
			ips, err := dns.ResolveAAAA(ctx, inTunl.LocalFQDN)
			if err != nil {
				l.Warn("LocalFQDN resolve failed", zap.String("fqdn", inTunl.LocalFQDN), zap.Error(err))
			} else {
				if len(ips) > 1 {
					l.Warn("LocalFQDN multiple AAAA records. Use first IPv6 address.", zap.String("fqdn", inTunl.LocalFQDN), zap.Strings("dns_answers", ips))
				}
				localIP = ips[0]
			}
		}
		// prefer *IPAddr field
		if inTunl.LocalIPAddr != "" {
			localIP = inTunl.LocalIPAddr
		}

		if inTunl.RemoteFQDN == "" && inTunl.RemoteIPAddr == "" {
			l.Error("RemoteFQDN and RemoteIPAddr are both empty", zap.String("tunnel", inTunl.Name))
			continue
		}
		if inTunl.RemoteFQDN != "" {
			ips, err := dns.ResolveAAAA(ctx, inTunl.RemoteFQDN)
			if err != nil {
				l.Warn("RemoteFQDN resolve failed", zap.String("fqdn", inTunl.RemoteFQDN), zap.Error(err))
			} else {
				if len(ips) > 1 {
					l.Warn("RemoteFQDN multiple AAAA records. Use first IPv6 address.", zap.String("fqdn", inTunl.RemoteFQDN), zap.Strings("dns_answers", ips))
				}
				remoteIP = ips[0]
			}
		}
		if inTunl.RemoteIPAddr != "" {
			remoteIP = inTunl.RemoteIPAddr
		}

		out.Tunnels = append(out.Tunnels, config.Tunnel{
			Name:         inTunl.Name,
			LocalIPAddr:  localIP,
			RemoteIPAddr: remoteIP,
		})
	}
	return &out, nil
}
