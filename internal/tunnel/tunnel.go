package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/enternejp/tap-etherip-config-sync/internal/config"
)

type TunnelConfig struct {
	Name     string
	LocalIP  string
	RemoteIP string
}

type Tunnel struct {
	EnvFileBasePath string
}

type TunnelAction string

const (
	ActionNoop     TunnelAction = "noop"
	ActionCreate   TunnelAction = "create"
	ActionDelete   TunnelAction = "delete"
	ActionRecreate TunnelAction = "recreate"
)

func (a TunnelAction) String() string {
	return string(a)
}

type Diff struct {
	Name   string
	Action TunnelAction
	Config *config.Tunnel
}

func (t *Tunnel) CreateOrRecreate(cfg TunnelConfig) error {
	envPath := filepath.Join(t.EnvFileBasePath, cfg.Name)
	f, err := os.Create(envPath)
	if err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "LOCAL=%s\nREMOTE=%s\n", cfg.LocalIP, cfg.RemoteIP)
	if err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	cmd := exec.Command("systemctl", "restart", "tap-etherip@"+cfg.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart systemd unit: %w", err)
	}
	return nil
}

func (t *Tunnel) Delete(name string) error {
	cmd := exec.Command("systemctl", "stop", "tap-etherip@"+name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop systemd unit: %w", err)
	}
	return nil
}

func (t *Tunnel) GetCurrentTunnels() ([]TunnelConfig, error) {
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--state=active", "--no-legend", "tap-etherip@*.service")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list systemd units: %w", err)
	}
	/*
		$ sudo systemctl list-units --type=service --state=active --no-legend "tap-etherip@*.service"
		tap-etherip@64496-1.service loaded active running Dummy EtherIP Service for E2E Test
	*/
	var result []TunnelConfig
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		unit := fields[0]
		name := unit[len("tap-etherip@") : len(unit)-len(".service")]
		envPath := filepath.Join(t.EnvFileBasePath, name)
		local, remote := readEnvFile(envPath)
		result = append(result, TunnelConfig{
			Name:     name,
			LocalIP:  local,
			RemoteIP: remote,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	return result, nil
}

func readEnvFile(path string) (string, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var local, remote string
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		if n, _ := fmt.Sscanf(line, "LOCAL=%s", &local); n == 1 {
			continue
		}
		if n, _ := fmt.Sscanf(line, "REMOTE=%s", &remote); n == 1 {
			continue
		}
	}
	return local, remote
}

func DiffTunnels(configs []config.Tunnel, current []TunnelConfig) []Diff {
	cfgMap := map[string]config.Tunnel{}
	for _, c := range configs {
		cfgMap[c.Name] = c
	}
	curMap := map[string]TunnelConfig{}
	for _, c := range current {
		curMap[c.Name] = c
	}

	var diffs []Diff
	for name, cfg := range cfgMap {
		cur, exists := curMap[name]
		if !exists {
			diffs = append(diffs, Diff{Name: name, Action: ActionCreate, Config: &cfg})
			continue
		}

		if cfg.LocalIPAddr != cur.LocalIP || cfg.RemoteIPAddr != cur.RemoteIP {
			diffs = append(diffs, Diff{Name: name, Action: ActionRecreate, Config: &cfg})
		} else {
			diffs = append(diffs, Diff{Name: name, Action: ActionNoop, Config: &cfg})
		}
	}

	for name := range curMap {
		if _, exists := cfgMap[name]; !exists {
			diffs = append(diffs, Diff{Name: name, Action: ActionDelete})
		}
	}
	return diffs
}
