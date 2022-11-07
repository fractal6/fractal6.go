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
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/tools"
)

var lang string
var creds model.UserCreds

var addUser = &cobra.Command{
    Use:   "adduser USERNAME EMAIL PASSWORD [--lang LANG]",
    Short: "Add an user to the database.",
    Long:  `Add an user to the databse.`,
    Args: cobra.MatchAll(
        cobra.MinimumNArgs(3),
        func(cmd *cobra.Command, args []string) error {
            creds = model.UserCreds{
                Username: args[0],
                Email: args[1],
                Password: args[2],
            }
            if l := cmd.Flag("lang"); l != nil {
                l := strings.ToUpper(l.Value.String())
                creds.Lang = &l
            }
            return auth.ValidateNewUser(creds)
    }),
    Run: func(cmd *cobra.Command, args []string) {
        AddUser(args)
    },
}

func init() {
    addUser.Flags().StringVar(&lang, "lang", "en", "User language.")
}

func AddUser(args []string) {
    creds.Password = tools.HashPassword(creds.Password)
    u, err := auth.CreateNewUser(creds)
    if err != nil {
        panic(err)
    } else if u == nil {
        fmt.Println("Something wen wrong, please check if user exists ?")
    } else {
        fmt.Println("User created.")
    }
}
