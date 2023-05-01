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

package tools

import (
	"fmt"
	"github.com/spf13/viper"
	"testing"
)

var matrixPostalRoom string
var matrixToken string
var DOMAIN string

func init() {
	InitViper()
	matrixPostalRoom = viper.GetString("mailer.matrix_postal_room")
	matrixToken = viper.GetString("mailer.matrix_token")
	DOMAIN = viper.GetString("server.domain")
}

func TestMatrixJsonSend(t *testing.T) {
	body := fmt.Sprintf(`"Hi! webhook test for %s"`, DOMAIN)
	err := MatrixJsonSend(string(body), matrixPostalRoom, matrixToken)
	if err != nil {
		t.Errorf("MatrixJsonSend failed: %s", err.Error())
	}
}
