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

package tools

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"
	//"io/ioutil"
	"github.com/spf13/viper"
)

type MatrixError struct {
	Errcode string `json:"errcode"`
	Error   string `json:"error"`
}

var MATRIX_DOMAIN string

func init() {
	InitViper()
	MATRIX_DOMAIN = viper.GetString("mailer.matrix_domain")
}

// Send a JSON formatted string to a matrix room
// @TODO: encryption: https://matrix.org/docs/guides/end-to-end-encryption-implementation-guide
func MatrixJsonSend(body, roomid, access_token string) error {
	// Pretiffy JSON string
	data, err := PrettyString(string(body))
	if err != nil {
		return err
	}

	// Send message to matrix room
	// @debug: triple backquote doesnt work with matrix; How to encode backquote ???
	//data = ```json\n" + QuoteString(data) + "\n```",
	data = QuoteString(data)
	data = fmt.Sprintf(`{
        "msgtype":"m.text",
        "body":"%s",
        "format": "org.matrix.custom.html",
        "formatted_body": "<pre><code class=\"language-json\">%s</code></pre>"
    }`, data, data)
	txnId := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	matrix_url := fmt.Sprintf(
		"https://%s/_matrix/client/r0/rooms/%s/send/m.room.message/%s?access_token=%s",
		MATRIX_DOMAIN,
		roomid,
		txnId,
		access_token)
	req, err := http.NewRequest("PUT", matrix_url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		//b, _ := ioutil.ReadAll(resp.Body)
		//fmt.Println(string(b))
		return fmt.Errorf("http matrix error, see body. (code %s)", resp.Status)
	}

	return nil
}
