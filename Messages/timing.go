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
	"image/color"
	"time"
)

type CarLocation int

// TODO - add garage and grid - need to calculate these based on speed and session type
const (
	NoLocation CarLocation = iota
	Pitlane
	PitOut
	OutLap
	OnTrack
	OutOfRace
	Stopped
)

func (c CarLocation) String() string {
	return [...]string{"Unknown", "Pitlane", "Pit Exit", "Out Lap", "On Track", "Out", "Stopped"}[c]
}

type TireType int

const (
	Unknown TireType = iota
	Soft
	Medium
	Hard
	Intermediate
	Wet
	Test
	HYPERSOFT
	ULTRASOFT
	SUPERSOFT
)

func (t TireType) String() string {
	return [...]string{"", "Soft", "Medium", "Hard", "Inter", "Wet", "Test", "Hyp Soft", "Ult Soft", "Sup Soft"}[t]
}

type SegmentType int

const (
	None SegmentType = iota
	YellowSegment
	GreenSegment
	InvalidSegment // Doesn't get displayed, cut corner/boundaries or invalid segment time?
	PurpleSegment
	RedSegment     // After chequered flag/stopped on track
	PitlaneSegment // In pitlane
	Mystery
	Mystery2 // ??? 2021 - Turkey Practice_2
	Mystery3 // ??? 2020 - Italy Race
)

type PitStop struct {
	Lap          int
	PitlaneEntry time.Time
	PitlaneExit  time.Time
	PitlaneTime  time.Duration
}

type Timing struct {
	Timestamp time.Time `json:"timestamp"`

	Position int `json:"position"`

	Name      string     `json:"name"`
	ShortName string     `json:"short_name"`
	Number    int        `json:"number"`
	Team      string     `json:"team"`
	HexColor  string     `json:"hex_color"`
	Color     color.RGBA `json:"color"`

	TimeDiffToFastest       time.Duration `json:"time_diff_to_fastest"`
	TimeDiffToPositionAhead time.Duration `json:"time_diff_to_position_ahead"`
	GapToLeader             time.Duration `json:"gap_to_leader"`

	PreviousSegmentIndex   int                      `json:"previous_segment_index"`
	Segment                [MaxSegments]SegmentType `json:"segment"`
	Sector1                time.Duration            `json:"sector1"`
	Sector1PersonalFastest bool                     `json:"sector1_personal_fastest"`
	Sector1OverallFastest  bool                     `json:"sector1_overall_fastest"`
	Sector2                time.Duration            `json:"sector2"`
	Sector2PersonalFastest bool                     `json:"sector2_personal_fastest"`
	Sector2OverallFastest  bool                     `json:"sector2_overall_fastest"`
	Sector3                time.Duration            `json:"sector3"`
	Sector3PersonalFastest bool                     `json:"sector3_personal_fastest"`
	Sector3OverallFastest  bool                     `json:"sector3_overall_fastest"`
	LastLap                time.Duration            `json:"last_lap"`
	LastLapPersonalFastest bool                     `json:"last_lap_personal_fastest"`
	LastLapOverallFastest  bool                     `json:"last_lap_overall_fastest"`

	FastestLap        time.Duration `json:"fastest_lap"`
	OverallFastestLap bool          `json:"overall_fastest_lap"`

	KnockedOutOfQualifying bool `json:"knocked_out_of_qualifying"`
	ChequeredFlag          bool `json:"chequered_flag"`

	Tire       TireType `json:"tire"`
	LapsOnTire int      `json:"laps_on_tire"`
	Lap        int      `json:"lap"`

	DRSOpen bool `json:"drs_open"`

	Pitstops     int       `json:"pitstops"`
	PitStopTimes []PitStop `json:"pit_stop_times"`

	Location CarLocation `json:"location"`

	SpeedTrap                int  `json:"speed_trap"`
	SpeedTrapPersonalFastest bool `json:"speed_trap_personal_fastest"`
	SpeedTrapOverallFastest  bool `json:"speed_trap_overall_fastest"`
}
