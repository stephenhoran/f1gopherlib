// F1GopherLib - Copyright (C) 2022 f1gopher
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package Messages

import (
	"time"
)

type Weather struct {
	Timestamp time.Time `json:"timestamp"`

	AirTemp       float64 `json:"air_temp"`
	Humidity      float64 `json:"humidity"`
	AirPressure   float64 `json:"air_pressure"`
	Rainfall      bool    `json:"rainfall"`
	TrackTemp     float64 `json:"track_temp"`
	WindDirection float64 `json:"wind_direction"`
	WindSpeed     float64 `json:"wind_speed"`
}
