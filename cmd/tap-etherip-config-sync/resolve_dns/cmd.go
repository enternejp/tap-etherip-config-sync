package resolve_dns

import (
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "resolve-dns",
		RunE: run,
	}
	return cmd
}
