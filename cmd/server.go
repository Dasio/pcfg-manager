package cmd

import (
	"github.com/dasio/pcfg-manager/server"
	"github.com/spf13/cobra"
)

var (
	inputArg server.InputArgs
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&inputArg.HashFile, "hashlist", "", "hash list to crack")
	serverCmd.Flags().StringVar(&inputArg.HashcatMode, "hashcatMode", "0", "hashcat mode of hash")

}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  "run server",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := server.NewService()
		inputArg.RuleName = ruleName
		if err := svc.Load(inputArg); err != nil {
			return err
		}
		return svc.Run()
	},
}
