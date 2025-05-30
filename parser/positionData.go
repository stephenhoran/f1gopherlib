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

package parser

import (
	"math"
	"strconv"
	"time"

	"github.com/stephenhoran/f1gopherlib/Messages"
	"github.com/stephenhoran/f1gopherlib/connection"
)

func (p *Parser) parsePositionData(dat map[string]interface{}, timestamp time.Time) ([]Messages.Location, error) {

	result := make([]Messages.Location, 0)
	const tolerance = 0.000001

	for _, record := range dat["Position"].([]interface{}) {
		timestampStr := record.(map[string]interface{})["Timestamp"].(string)
		dataTimestamp, err := parseTime(timestampStr)
		if err != nil {
			p.ParseTimeError(connection.PositionFile, timestamp, "Timestamp", err)
		}

		for key, entry := range record.(map[string]interface{})["Entries"].(map[string]interface{}) {
			driver, _ := strconv.ParseInt(key, 10, 8)
			//status := entry.(map[string]interface{})["Status"].(string)

			x := entry.(map[string]interface{})["X"].(float64)
			y := entry.(map[string]interface{})["Y"].(float64)
			z := entry.(map[string]interface{})["Z"].(float64)

			// Ignore locations which are (0, 0) because it means we don't have a location for them
			if math.Abs(x) < tolerance && math.Abs(y) < tolerance {
				continue
			}

			result = append(result, Messages.Location{
				Timestamp:    dataTimestamp,
				DriverNumber: int(driver),
				X:            x,
				Y:            y,
				Z:            z,
			})
		}
	}

	return result, nil
}
