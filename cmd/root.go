
package cmd

import (
	//"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"fractale/fractal6.go/tools"
)

var (
	rootCmd = &cobra.Command{
		Use:   "fractal6",
		Short: "Self organisation platform for humans.",
		Long:  `Self organisation platform for humans.`,
	}
)

var apiCmd = &cobra.Command{
    Use:   "api",
    Short: "run server.",
    Long:  `run server.`,
    Run: func(cmd *cobra.Command, args []string) {
        RunServer()
    },
}

var notifierCmd = &cobra.Command{
    Use:   "notifier",
    Short: "run notifier daemon.",
    Long:  `run notifier daemon.`,
    Run: func(cmd *cobra.Command, args []string) {
        RunNotifier()
    },
}

func init() {
	cobra.OnInitialize(tools.InitViper)
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

    // Cli init
    rootCmd.AddCommand(apiCmd)
    rootCmd.AddCommand(notifierCmd)
}

// Run the root command.
func Run() {
	if err := rootCmd.Execute(); err != nil {
        //fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}



