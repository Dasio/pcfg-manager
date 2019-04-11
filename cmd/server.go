package cmd

import (
	"github.com/dasio/pcfg-manager/manager"
	"github.com/dasio/pcfg-manager/server"
	"github.com/spf13/cobra"
	"time"
)

var (
	serverArgs manager.InputArgs
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&serverArgs.HashFile, "hashlist", "", "hash list to crack")
	serverCmd.Flags().StringVar(&serverArgs.HashcatMode, "hashcatMode", "0", "hashcat mode of hash")
	serverCmd.Flags().StringVarP(&serverArgs.Port, "port", "p", "50051", "server port")
	serverCmd.Flags().Uint64VarP(&serverArgs.MaxGuesses, "maxGuesses", "m", 0, "max guesses before exit")
	serverCmd.Flags().IntVar(&serverArgs.TerminalsQueSize, "termQueSize", 100000, "how many pre-terminals structure leads to terminals can be in que - 0 for unlimited")
	serverCmd.Flags().Uint64Var(&serverArgs.ChunkStartSize, "chunkStartSize", 10000, "how many pre-terminals will be sent at connected client")
	serverCmd.Flags().DurationVar(&serverArgs.ChunkDuration, "chunkDuration", time.Second*30, "how long should each chunk take")

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
