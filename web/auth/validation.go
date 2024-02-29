/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
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

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"strings"
	"time"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/tools"
)

var clientVersion string
var reservedUsername map[string]bool
var MAX_PUBLIC_ORGA int
var MAX_PRIVATE_ORGA int
var MAX_ORGA_REG int
var MAX_ORGA_PRO int

func init() {
	var err error
	clientVersion = viper.GetString("server.client_version")
	MAX_PUBLIC_ORGA, err = strconv.Atoi(viper.GetString("admin.max_public_orgas"))
	if err != nil {
		fmt.Println("max_public_orgas conf not found, setting to 100")
		MAX_PUBLIC_ORGA = 100
	}
	MAX_PRIVATE_ORGA, err = strconv.Atoi(viper.GetString("admin.max_private_orgas"))
	if err != nil {
		fmt.Println("max_private_orgas conf not found, setting to 100")
		MAX_PRIVATE_ORGA = 100
	}
	MAX_ORGA_REG, err = strconv.Atoi(viper.GetString("admin.max_orga_reg"))
	if err != nil {
		fmt.Println("max_orga_reg conf not found, setting to 100")
		MAX_ORGA_REG = 100
	}
	MAX_ORGA_PRO, err = strconv.Atoi(viper.GetString("admin.max_orga_pro"))
	if err != nil {
		fmt.Println("max_orga_pro conf not found, setting to 100")
		MAX_ORGA_PRO = 100
	}
	reservedUsername = map[string]bool{
		// Reserved email endpoint
		"root":          true,
		"admin":         true,
		"sysadmin":      true,
		"alert":         true,
		"contact":       true,
		"security":      true,
		"notifications": true,
		"noreply":       true,
		"dmarc-reports": true,
		"postmaster":    true,
		// Reserved URI
		// --
		// back
		"ping":           true,
		"playground":     true,
		"metrics":        true,
		"mailing":        true,
		"postal_webhook": true,
		"api":            true,
		"auth":           true,
		"data":           true,
		"assets":         true,
		"static":         true,
		"index":          true,
		"index.html":     true,
		"ghost":          true,
		// front - about
		"about":            true,
		"help":             true,
		"features":         true,
		"showcases":        true,
		"marketplace":      true,
		"solutions":        true,
		"pricing":          true,
		"trending":         true,
		"topics":           true,
		"manifesto":        true,
		"sitemap":          true,
		"enterprise":       true,
		"collective":       true,
		"developers":       true,
		"terms-of-service": true,
		// front - app
		"home":           true,
		"dashboard":      true,
		"new":            true, // tension, orga, networks
		"explore":        true, // orgas, networks, users
		"login":          true,
		"logout":         true,
		"signup":         true,
		"verification":   true,
		"password-reset": true,
		"user":           true,
		"users":          true,
		"tension":        true,
		"tensions":       true,
		"project":        true,
		"projects":       true,
		"calendar":       true,
		"calendars":      true,
		"org":            true,
		"orgs":           true,
		"network":        true,
		"networks":       true,
	}
}

func regularizeUctx(uctx *model.UserCtx) {
	// Hide the password !
	uctx.Password = ""
	// Set the client version
	uctx.ClientVersion = clientVersion
	// Set the date of expiration (based on the jwt token validity)
	uctx.ExpiresAt = time.Now().Add(tokenValidityTime).UTC().Format(time.RFC3339)
}

//
// Public methods
//

// GetUser returns the user ctx from a db.grpc request,
// **if they are authenticated** against their hashed password.
// Note: ===  This is protected ===
// This method check if the user has login rights, and check passwords.
func GetAuthUserCtx(creds model.UserCreds) (*model.UserCtx, error) {
	// 1. get username/email or throw error
	// 3. if pass compare pasword or throw error
	// 4. if pass, returns UsertCtx from db request or throw error
	var fieldId string
	var userId string

	username := creds.Username
	password := creds.Password

	// Validate signin form
	err := ValidateSimplePassword(password)
	if err != nil {
		return nil, err
	} else if len(username) > 1 {
		if strings.Contains(username, "@") {
			fieldId = "email"
		} else {
			fieldId = "username"
		}
		userId = username
	} else {
		return nil, ErrBadUsernameFormat
	}

	// Try getting usetCtx
	userCtx, err := db.GetDB().GetUctx(fieldId, userId)
	if err != nil {
		return nil, FormatError(err, "fieldid")
	}

	// Check if the user has login authorization
	if !userCtx.Rights.CanLogin {
		return nil, ErrCantLogin
	}

	// Compare hashed password.
	ok := tools.VerifyPassword(userCtx.Password, password)
	if !ok {
		return nil, ErrWrongPassword
	}

	regularizeUctx(userCtx)
	return userCtx, nil
}

// GetAuthUserFromCtx returns the user ctx from a db.grpc request,
// from the given user context.
func GetAuthUserFromCtx(uctx model.UserCtx) (*model.UserCtx, error) {
	// Try getting userCtx
	userCtx, err := db.GetDB().GetUctx("username", uctx.Username)
	if err != nil {
		return nil, FormatError(err, "username")
	}

	// Check if the user has login authorization
	if !userCtx.Rights.CanLogin {
		return nil, ErrCantLogin
	}

	// Update the user roles cache.
	ctx := context.Background()
	var key string = userCtx.Username + "roles"
	d, _ := json.Marshal(userCtx.Roles)
	err = cache.SetEX(ctx, key, d, time.Second*12).Err()
	if err != nil {
		return nil, FormatError(err, "")
	}

	regularizeUctx(userCtx)
	return userCtx, nil
}

// ValidateNewuser check that an user doesn't exist,
// from a db.grpc request.
func ValidateNewUser(creds model.UserCreds) error {
	username := creds.Username
	email := creds.Email
	name := creds.Name
	lang := creds.Lang
	password := creds.Password

	// Username validation
	err := ValidateUsername(username)
	if err != nil {
		return err
	} else if reservedUsername[username] {
		return ErrUsernameExist
	}
	// Email validation
	err = ValidateEmail(email)
	if err != nil {
		return err
	}
	// Name validation
	if name != nil {
		err = ValidateName(*name)
		if err != nil {
			return err
		}
	}
	// Lang validation
	if lang != nil {
		if !model.Lang(*lang).IsValid() {
			return fmt.Errorf("Bad value for lang.")
		}
	}

	// Password validation
	err = ValidatePassword(password)
	if err != nil {
		return err
	}
	// TODO: password complexity check

	// Check username existence
	ex1, err1 := db.DB.Exists("User.username", username, nil)
	if err1 != nil {
		return err1
	}
	if ex1 {
		return ErrUsernameExist
	}
	// Check email existence
	ex2, err2 := db.DB.Exists("User.email", email, nil)
	if err2 != nil {
		return err2
	}
	if ex2 {
		return ErrEmailExist
	}

	// New user can be created !
	return nil
}

// CreateNewUser Upsert an user,
// using db.graphql request.
func CreateNewUser(creds model.UserCreds) (*model.UserCtx, error) {
	now := tools.Now()
	// Rights
	userType := model.UserTypeRegular
	maxPublicOrga := MAX_PUBLIC_ORGA
	maxPrivateOrga := MAX_PRIVATE_ORGA
	hasEmailNotifications := true
	canLogin := true
	canCreateRoot := false
	lang := model.LangEn

	if creds.CanLogin != nil {
		canLogin = bool(*creds.CanLogin)
	}
	if creds.Lang != nil {
		lang = model.Lang(*creds.Lang)
	}

	userInput := model.AddUserInput{
		CreatedAt:     now,
		LastAck:       now,
		NotifyByEmail: true,
		Lang:          lang,
		Username:      creds.Username,
		Email:         creds.Email,
		Name:          creds.Name,
		Password:      creds.Password,
		Rights: &model.UserRightsRef{
			CanLogin:              &canLogin,
			CanCreateRoot:         &canCreateRoot,
			MaxPublicOrga:         &maxPublicOrga,
			MaxPrivateOrga:        &maxPrivateOrga,
			Type:                  &userType,
			HasEmailNotifications: &hasEmailNotifications,
		},
	}

	_, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "user", userInput)
	if err != nil {
		return nil, err
	}

	// Try getting userCtx
	userCtx, err := db.GetDB().GetUctx("username", creds.Username)
	if err != nil {
		return nil, err
	}

	regularizeUctx(userCtx)
	return userCtx, nil
}

//
// Verify New orga right
//

func CanNewOrga(uctx model.UserCtx, form model.OrgaForm) (bool, error) {
	var ok bool
	var err error

	regex := fmt.Sprintf("@%s$", uctx.Username)
	nodes, err := db.GetDB().GetNodes(regex, true)
	if err != nil {
		return ok, err
	}
	n_public := 0
	n_private := 0
	for _, n := range nodes {
		if n.Visibility == model.NodeVisibilityPublic {
			n_public += 1
		} else if n.Visibility == model.NodeVisibilityPrivate {
			n_private += 1
		}
	}
	n_orgs := len(nodes)

	switch uctx.Rights.Type {
	case model.UserTypeRegular:
		if n_orgs >= MAX_ORGA_REG && MAX_ORGA_REG >= 0 {
			return ok, fmt.Errorf("Number of organisation are limited to %d, please contact us to create more.", MAX_ORGA_REG)
		} else if *form.Visibility == model.NodeVisibilityPublic &&
			n_public >= uctx.Rights.MaxPublicOrga && uctx.Rights.MaxPublicOrga >= 0 {
			return ok, fmt.Errorf("Number of public organisation are limited to %d, please contact us to create more.", uctx.Rights.MaxPublicOrga)
		} else if *form.Visibility == model.NodeVisibilityPrivate &&
			n_private >= uctx.Rights.MaxPrivateOrga && uctx.Rights.MaxPrivateOrga >= 0 {
			return ok, fmt.Errorf("Number of private organisation are limited to %d, please contact us to create more.", uctx.Rights.MaxPrivateOrga)
		}

	case model.UserTypePro:
		if n_orgs >= MAX_ORGA_PRO && MAX_ORGA_PRO >= 0 {
			return ok, fmt.Errorf("Number of organisation are limited to %d, please contact us to create more.", MAX_ORGA_PRO)
		}

	case model.UserTypeRoot:
		// pass
	}

	ok = true
	return ok, err
}
