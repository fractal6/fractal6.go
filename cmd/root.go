
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"fractale/fractal6.go/tools"
)

var (
	rootCmd = &cobra.Command{
		Use:   "fractal6",
		Short: "Self organisation platform for humans.",
		Long: `FIXME`,
	}
)

// Run the root command.
func Run() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(tools.InitViper)
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

}

