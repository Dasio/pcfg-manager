package cmd

import (
	"github.com/dasio/pcfg-manager/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var (
	clientArgs client.InputArgs
)

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.Flags().StringVarP(&clientArgs.ServerAddress, "server", "s", "localhost:50051", "server address")
	clientCmd.Flags().StringVar(&clientArgs.HashcatFolder, "hashcatFolder", "./hashcat", "folder in which is hashcat binary")
	clientCmd.Flags().BoolVar(&clientArgs.GenOnly, "generateOnly", false, "generation guesses without cracking")

}

var clientCmd = &cobra.Command{
	Use:           "client",
	Short:         "run client",
	Long:          "run client",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := client.NewService(clientArgs)
		if err != nil {
			return err
		}
		if err := svc.Connect(clientArgs.ServerAddress); err != nil {
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
