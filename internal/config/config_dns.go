package config

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

type ConfWithDNS struct {
	Tunnels []TunnelWithDNS `json:"tunnels"`
}

type TunnelWithDNS struct {
	Name         string `json:"name"`
	LocalIPAddr  string `json:"local_ip_addr"`
	LocalFQDN    string `json:"local_fqdn"`
	RemoteIPAddr string `json:"remote_ip_addr"`
	RemoteFQDN   string `json:"remote_fqdn"`
}

func NewWithDNS(reader io.Reader) (*ConfWithDNS, error) {
	conf := ConfWithDNS{}
	if err := json.NewDecoder(reader).Decode(&conf); err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	return &conf, nil
}
