package cmd

import (
	"github.com/dasio/pcfg-manager/manager"
	"github.com/spf13/cobra"
	"os"
	"runtime/pprof"
)

var (
	ruleName   string
	cpuProfile string
)
var rootCmd = &cobra.Command{
	Use:   "pcfg-manager",
	Short: "Password generator",
	Long:  `Password generator`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				return err
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				return err
			}
			defer pprof.StopCPUProfile()
		}
		mng := manager.NewManager(ruleName)
		if err := mng.Load(); err != nil {
			return err
		}
		if err := mng.Start(); err != nil {
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
	rootCmd.PersistentFlags().StringVarP(&cpuProfile, "cpuprofile", "c", "", "write cpu profile to file")

}

func initConfig() {
}
