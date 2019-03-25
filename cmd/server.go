package cmd

import (
	"github.com/dasio/pcfg-manager/server"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  "run server",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := server.NewService()
		if err := svc.Load(ruleName); err != nil {
			return err
		}
		return svc.Run()
	},
}
