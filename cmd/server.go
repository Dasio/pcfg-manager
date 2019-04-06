package cmd

import (
	"github.com/dasio/pcfg-manager/server"
	"github.com/spf13/cobra"
)

var (
	serverArgs server.InputArgs
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&serverArgs.HashFile, "hashlist", "", "hash list to crack")
	serverCmd.Flags().StringVar(&serverArgs.HashcatMode, "hashcatMode", "0", "hashcat mode of hash")

}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  "run server",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := server.NewService()
		serverArgs.RuleName = ruleName
		if err := svc.Load(serverArgs); err != nil {
			return err
		}
		return svc.Run()
	},
}
