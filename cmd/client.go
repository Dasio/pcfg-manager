package cmd

import (
	"github.com/dasio/pcfg-manager/client"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "run client",
	Long:  "run client",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := client.NewService()
		if err := svc.Connect("localhost:50051"); err != nil {
			return err
		}
		return svc.Disconnect()
	},
}
