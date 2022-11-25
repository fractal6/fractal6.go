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
    "reflect"
    "testing"
	"encoding/json"
)

func TestUserCreds(t *testing.T) {
    testcases := []struct{
        input string
        want UserCreds
    }{
        {"{}", UserCreds{}},
        {`{"username":"coucou"}`, UserCreds{Username:"coucou"}},
        {`{"username":"coucou", "subscribe":true}`, UserCreds{Username:"coucou", Subscribe: true}},
        {`{"username":"coucou", "subscribe":false}`, UserCreds{Username:"coucou", Subscribe: false}},
        {`{"username":"coucou", "subscribe":"true"}`, UserCreds{Username:"coucou", Subscribe: true}},
        {`{"username":"coucou", "subscribe":"false"}`, UserCreds{Username:"coucou", Subscribe: false}},
        {`{"username":"coucou", "subscribe":"1"}`, UserCreds{Username:"coucou", Subscribe: true}},
        {`{"username":"coucou", "subscribe":"0"}`, UserCreds{Username:"coucou", Subscribe: false}},
        {`{"username":"coucou", "subscribe":1}`, UserCreds{Username:"coucou", Subscribe: true}},
        {`{"username":"coucou", "subscribe":0}`, UserCreds{Username:"coucou", Subscribe: false}},
        {`{"username":"coucou", "subscribe":"other"}`, UserCreds{Username:"coucou", Subscribe: false}},
	}

    for _, test := range testcases {
        //t.Logf("testcase %d", i)
        //b, _ := json.Marshal(test.input)
        b := []byte(test.input)

        var got UserCreds
        err := json.Unmarshal(b, &got)
        if err != nil {
            t.Errorf(err.Error())
        }

        if !reflect.DeepEqual(got, test.want) {
            t.Errorf("For p = %s, want %v. Got %v",
            test.input, test.want, got)
        }
    }
}

func TestUserCredsMap(t *testing.T) {
    type Pending struct{
        Username string
        Email string
        Password string
        UpdatedAt *string
        Subscribe  bool
    }

    testcases := []struct{
        input Pending
        want UserCreds
    }{
        {Pending{Username:"coucou"}, UserCreds{Username:"coucou"}},
        {Pending{Username:"coucou", Subscribe: true}, UserCreds{Username:"coucou", Subscribe:true}},
        {Pending{Username:"coucou", Subscribe: false}, UserCreds{Username:"coucou", Subscribe:false}},
	}

    for _, test := range testcases {
        var got UserCreds
        StructMap(test.input, &got)
        if !reflect.DeepEqual(got, test.want) {
            t.Errorf("For p = %v, want %v. Got %v",
            test.input, test.want, got)
        }
    }
}
