
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	cobra.OnInitialize(initConfig)

    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

}


func initConfig() {

	configName := "config"
	
	viper.SetConfigName(configName) // name of config file (without extension)
	viper.AddConfigPath(".")
	// 	viper.AutomaticEnv() // read in environment variables that match

    // If a config file is found, read it in.
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    } else {
        fmt.Println(err)
    }

}
