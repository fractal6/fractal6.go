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

package model

import (
    "log"
    "strconv"
    "strings"
)

// Type boolean unmarshal string (true or false) to a bool like value.
type Boolean bool

func (bit *Boolean) UnmarshalJSON(data []byte) error {
    asString := strings.Trim(string(data), "\"")
    res, err := strconv.ParseBool(asString)
    if err != nil {
        log.Printf("Boolean unmarshal error: invalid input %s; %s", asString, err.Error())
        *bit = false
    }
    if res {
        *bit = true
    } else {
        *bit = false
    }
    return nil
}

func (bit Boolean) ToBoolPtr() *bool {
    a := bool(bit)
    return &a
}
