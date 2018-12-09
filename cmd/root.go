package cmd

import (
	"github.com/dasio/pcfg-manager/manager"
	"github.com/spf13/cobra"
)

var (
	ruleName   string
	goRoutines uint
	maxGuesses uint64
)
var rootCmd = &cobra.Command{
	Use:   "pcfg-manager",
	Short: "Password generator",
	Long:  `Password generator`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mng := manager.NewManager(ruleName)
		if err := mng.Load(); err != nil {
			return err
		}
		if err := mng.Start(goRoutines, maxGuesses); err != nil {
			return err
		}
		return nil
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&ruleName, "rule-name", "r", "Default", "specifies rule")
	rootCmd.PersistentFlags().UintVarP(&goRoutines, "go-routines", "g", 1, "how many go routines will be used")
	rootCmd.PersistentFlags().Uint64VarP(&maxGuesses, "max-guesses", "m", 0, "max guesses before exit")

}

func initConfig() {
}
