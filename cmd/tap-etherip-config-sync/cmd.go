package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/enternejp/tap-etherip-config-sync/cmd/tap-etherip-config-sync/resolve_dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagKeyConfig   = "config"
	flagLogLevel    = "log-level"
	flagEnvBasePath = "env-base-path"
)

func main() {
	var rootCmd = &cobra.Command{
		Use: "tap-etherip-config-sync",
		// PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 	perfListenAddr, err := cmd.Flags().GetString("perf-listen-addr")
		// 	if err != nil {
		// 		return err
		// 	}
		// 	http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())
		// 	go func() {
		// 		fmt.Println(http.ListenAndServe(perfListenAddr, nil))
		// 	}()
		// 	return nil
		// },
		RunE: run,
	}
	// rootCmd.PersistentFlags().String("perf-listen-addr", "127.0.0.1:6060", "pprof listen address")
	rootCmd.Flags().String(flagLogLevel, "info", "log level")
	rootCmd.Flags().String(flagKeyConfig, "./config.json", "config file path")
	rootCmd.Flags().String(flagEnvBasePath, "./tap-etherip-envs", "env file base path")

	viper.SetEnvPrefix("TAP_ETHERIP_CONFIG_SYNC")
	viper.BindPFlags(rootCmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	rootCmd.AddCommand(resolve_dns.Cmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}
