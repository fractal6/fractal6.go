/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
	"github.com/spf13/cobra"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

var lang string
var creds model.UserCreds

var addUser = &cobra.Command{
	Use:   "adduser USERNAME EMAIL PASSWORD [--lang LANG]",
	Short: "Add an user to the database",
	Long:  `Add an user to the databse.`,
	Args: cobra.MatchAll(
		cobra.MinimumNArgs(3),
		// Check that credentials are valid.
		func(cmd *cobra.Command, args []string) error {
			creds = model.UserCreds{
				Username: args[0],
				Email:    args[1],
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

var delUser = &cobra.Command{
	Use:   "deluser USERNAME",
	Short: "Delete an user from the database",
	Long:  `Delete an user from the database.`,
	Args: cobra.MatchAll(
		cobra.MinimumNArgs(1),
	),
	Run: func(cmd *cobra.Command, args []string) {
		DelUser(args)
	},
}

func init() {
	addUser.Flags().StringVar(&lang, "lang", "en", "User language (en, fr).")
}

func AddUser(args []string) {
	creds.Password = tools.HashPassword(creds.Password)
	u, err := auth.CreateNewUser(creds)
	if err != nil {
		panic(err)
	} else if u == nil {
		fmt.Println("Something went wrong, please check if user has been created?")
	} else {
		fmt.Println("User created.")
	}
}

func DelUser(args []string) {
	username := args[0]
	u, err := db.GetDB().GetFieldByEq("User.username", username, "uid")
	if err != nil {
		panic(err)
	} else if u == nil {
		fmt.Printf("User '%s' not found.\n", username)
		return
	}

	// If ghost user has not been created yet, create it
	var canLogin model.Boolean = false
	name := "Deleted user"
	ghost := model.UserCreds{Username: "ghost", Email: "ghost@fractale.co", Name: &name, CanLogin: &canLogin}
	g, err := db.GetDB().GetFieldByEq("User.username", "ghost", "uid")
	if err != nil {
		panic(err)
	} else if g == nil {
		g, err = auth.CreateNewUser(ghost)
		if err != nil {
			panic(err)
		}
		about := `Hi, I'm @ghost! I take the place of user accounts that have been deleted. 👻`
		db.GetDB().SetFieldByEq("User.username", ghost.Username, "User.about", about)
		fmt.Println("ghost created")
	}

	g, err = db.GetDB().GetFieldByEq("User.username", "ghost", "uid")
	if err != nil {
		panic(err)
	}
	ghostid := g.(string)

	// Deep delete an user:
	// - Clean user orphan data
	// - Replace all createdBy field by ghost (del old_user + and ghost)
	_, err = db.GetDB().Meta("deleteUser", map[string]string{"username": username, "ghostid": ghostid})
	if err != nil {
		panic(err)
	}

	fmt.Printf("User '%s' has been deleted.\n", username)
}
