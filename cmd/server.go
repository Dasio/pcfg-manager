package cmd

import (
	"github.com/dasio/pcfg-manager/server"
	"github.com/spf13/cobra"
)

var (
	hashFile string
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().StringVar(&hashFile, "hashlist", "", "hash list to crack")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  "run server",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := server.NewService()
		if err := svc.Load(ruleName, hashFile); err != nil {
			return err
		}
		return svc.Run()
	},
}
