/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package cmd

import (
	//"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

var (
    rootCmd = &cobra.Command{
        Use:   "f6",
        Short: "Fractale - Self-organisation for humans",
        Long:  `Fractale - Self-organisation for humans.`,
    }

    apiCmd = &cobra.Command{
        Use:   "api",
        Short: "run server",
        Long:  `run server.`,
        PreRun: func(cmd *cobra.Command, args []string) {
            viper.SetDefault("rootCmd", "api")
        },
        Run: func(cmd *cobra.Command, args []string) {
            RunServer()
        },
    }

    notifierCmd = &cobra.Command{
        Use:   "notifier",
        Short: "run notifier daemon",
        Long:  `run notifier daemon.`,
        Run: func(cmd *cobra.Command, args []string) {
            RunNotifier()
        },
    }

    genToken = &cobra.Command{
        Use:   "token",
        Short: "Generate JWT tokens",
        Long:  `Generate JWT tokens.`,
        Run: func(cmd *cobra.Command, args []string) {
            auth.GenToken()
        },
    }

)

func init() {
	cobra.OnInitialize(tools.InitViper)
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("rootCmd", rootCmd.PersistentFlags().Lookup("verbose"))

    // Cli init
    rootCmd.AddCommand(apiCmd)
    rootCmd.AddCommand(notifierCmd)
    rootCmd.AddCommand(genToken)
    rootCmd.AddCommand(addUser)
}

// Run the root command.
func Run() {
	if err := rootCmd.Execute(); err != nil {
        //fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}



