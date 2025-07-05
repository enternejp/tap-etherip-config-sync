package config

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

type Conf struct {
	Tunnels []Tunnel `json:"tunnels"`
}

type Tunnel struct {
	Name         string `json:"name"`
	LocalIPAddr  string `json:"local_ip_addr"`
	RemoteIPAddr string `json:"remote_ip_addr"`
}

func New(reader io.Reader) (*Conf, error) {
	conf := Conf{}
	if err := json.NewDecoder(reader).Decode(&conf); err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	return &conf, nil
}
