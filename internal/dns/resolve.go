package dns

import (
	"context"
	"fmt"
	"net"
)

func ResolveAAAA(ctx context.Context, fqdn string) ([]string, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", fqdn)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve AAAA for %s: %w", fqdn, err)
	}
	var results []string
	for _, ip := range ips {
		if ip.To16() != nil && ip.To4() == nil {
			results = append(results, ip.String())
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no AAAA record found for %s", fqdn)
	}
	return results, nil
}
