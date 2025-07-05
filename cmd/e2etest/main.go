package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cfg "github.com/enternejp/tap-etherip-config-sync/internal/config"
)

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func main() {
	serviceSrc := "test/e2e/tap-etherip@.service"
	serviceDst := "/etc/systemd/system/tap-etherip@.service"
	if err := copyFile(serviceSrc, serviceDst); err != nil {
		fmt.Fprintf(os.Stderr, "failed to copy service file: %v\n", err)
		os.Exit(1)
	}
	cmdReload := exec.Command("systemctl", "daemon-reload")
	cmdReload.Stdout = os.Stdout
	cmdReload.Stderr = os.Stderr
	if err := cmdReload.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run systemctl daemon-reload: %v\n", err)
		os.Exit(1)
	}

	configDir := "testdata/configs"
	configs, err := filepath.Glob(filepath.Join(configDir, "*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list config files: %v\n", err)
		os.Exit(1)
	}
	if len(configs) == 0 {
		fmt.Fprintf(os.Stderr, "no config files found in %s\n", configDir)
		os.Exit(1)
	}

	envBasePath := "/tmp/tap-etherip"
	if err := os.RemoveAll(envBasePath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", envBasePath, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(envBasePath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create %s: %v\n", envBasePath, err)
		os.Exit(1)
	}

	_ = exec.Command("ip", "netns", "del", "tunnel1").Run() // ensure recreate
	if err := exec.Command("ip", "netns", "add", "tunnel1").Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create netns tunnel1: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := exec.Command("ip", "netns", "del", "tunnel1").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to delete netns tunnel1: %v\n", err)
		}
	}()

	sleepSrc := "test/e2e/sleep-infinity.sh"
	sleepDst := filepath.Join(envBasePath, "sleep-infinity.sh")
	if err := copyFile(sleepSrc, sleepDst); err != nil {
		fmt.Fprintf(os.Stderr, "failed to copy sleep-infinity.sh: %v\n", err)
		os.Exit(1)
	}
	if err := os.Chmod(sleepDst, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to chmod sleep-infinity.sh: %v\n", err)
		os.Exit(1)
	}

	allPassed := true
	for _, config := range configs {
		fmt.Printf("=== Testing... config=%s ===\n", config)

		f, err := os.Open(config)
		if err != nil {
			fmt.Printf("FAILED: could not open config: %v\n", err)
			allPassed = false
			continue
		}
		defer f.Close()
		conf, err := cfg.New(f)
		if err != nil {
			fmt.Printf("FAILED: could not parse config: %v\n", err)
			allPassed = false
			continue
		}

		cmd := exec.Command("./tap-etherip-config-sync", "--log-level", "debug", "--config", config, "--env-base-path", envBasePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Printf("FAILED: tap-etherip-config-sync exited with error: %v\n", err)
			allPassed = false
			continue
		}

		pass := true
		psCmd := exec.Command("bash", "-c", "ps -eo pid,args | grep -i sleep-infinity.sh | grep -v grep")
		psOut, err := psCmd.Output()
		if err != nil && len(conf.Tunnels) > 0 {
			fmt.Printf("FAILED: could not check processes: %v\n", err)
			allPassed = false
			continue
		}
		psLines := strings.Split(string(psOut), "\n")
		log.Printf("ps output:\n%s", psOut)

		found := map[string]bool{}
		for _, tun := range conf.Tunnels {
			found[tun.Name] = false
		}
		for _, line := range psLines {
			if !strings.Contains(line, "sleep-infinity.sh") {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}

			// <PID> sleep-infinity.sh <IF_NAME> <LOCAL> <REMOTE>
			name := fields[len(fields)-3]
			local := fields[len(fields)-2]
			remote := fields[len(fields)-1]
			for _, tun := range conf.Tunnels {
				if tun.Name == name && tun.LocalIPAddr == local && tun.RemoteIPAddr == remote {
					found[name] = true
				}
			}
		}
		for _, tun := range conf.Tunnels {
			if !found[tun.Name] {
				fmt.Printf("FAILED: process for tunnel %s (LOCAL=%s REMOTE=%s) not found in ps\n", tun.Name, tun.LocalIPAddr, tun.RemoteIPAddr)
				pass = false
			}
		}

		if len(conf.Tunnels) == 0 {
			for _, line := range psLines {
				if strings.Contains(line, "sleep-infinity.sh") {
					fmt.Printf("FAILED: unexpected sleep-infinity.sh process found: %s\n", line)
					pass = false
				}
			}
		}

		if pass {
			fmt.Printf("PASS: All tunnels and processes match for %s\n", config)
		} else {
			allPassed = false
		}
	}

	if allPassed {
		fmt.Println("all pass")
		os.Exit(0)
	}

	fmt.Println("failed")
	os.Exit(1)
}
