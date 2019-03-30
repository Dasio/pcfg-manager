package cmd

import (
	"github.com/dasio/pcfg-manager/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:           "client",
	Short:         "run client",
	Long:          "run client",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := client.NewService()
		if err := svc.Connect("localhost:50051"); err != nil {
			return err
		}
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			done <- true
		}()
		if err := svc.Run(done); err != nil {
			logrus.Warn(err)
		}
		return svc.Disconnect()
	},
}
