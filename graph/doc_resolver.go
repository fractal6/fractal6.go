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

package graph

import (
	"fractale/fractal6.go/graph/model"
)

func TryAddDoc(uctx *model.UserCtx, tension *model.Tension, md *string) (bool, error) {
	return false, nil
}

func TryUpdateDoc(uctx *model.UserCtx, tension *model.Tension, md *string) (bool, error) {
	return false, nil
}

func TryChangeArchiveDoc(uctx *model.UserCtx, tension *model.Tension, md *string, event model.TensionEvent) (bool, error) {
	return false, nil
}
