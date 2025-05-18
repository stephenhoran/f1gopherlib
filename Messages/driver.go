package Messages

import (
	"image/color"
	"time"
)

type DriverInfo struct {
	StartPosition int        `json:"start_position"`
	Name          string     `json:"name"`
	ShortName     string     `json:"short_name"`
	Number        int        `json:"number"`
	Team          string     `json:"team"`
	HexColor      string     `json:"hex_color"`
	Color         color.RGBA `json:"color"`
}

type Drivers struct {
	Timestamp time.Time `json:"timestamp"`

	Drivers []DriverInfo `json:"drivers"`
}
