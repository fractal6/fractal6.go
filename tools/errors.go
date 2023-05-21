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
	"fmt"
	"log"
	"runtime"
)

func LogErr(reason string, err error) error {
	// Get trace information
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)    // Skip 2 levels to get the caller
	f := runtime.FuncForPC(pc[0])
	fname := f.Name()
	//file, line := f.FileLine(pc[0])

	log.Printf("[@%s] %s: %s", fname, reason, err.Error())
	return fmt.Errorf("%s: %s", reason, err.Error())
}
