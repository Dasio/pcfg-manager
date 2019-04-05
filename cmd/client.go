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
	hashcatFolder string
	serverAddress string
)

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.Flags().StringVarP(&serverAddress, "server", "s", "localhost:50051", "server address")
	clientCmd.Flags().StringVar(&hashcatFolder, "hashcatFolder", "./hashcat", "folder in which is hashcat binary")
}

var clientCmd = &cobra.Command{
	Use:           "client",
	Short:         "run client",
	Long:          "run client",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := client.NewService(hashcatFolder)
		if err != nil {
			return err
		}
		if err := svc.Connect(serverAddress); err != nil {
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
