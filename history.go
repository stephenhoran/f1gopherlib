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

package f1gopherlib

import (
	"strings"
	"time"

	"github.com/stephenhoran/f1gopherlib/Messages"
)

func RaceHistory() []RaceEvent {
	result := make([]RaceEvent, 0)

	for _, session := range sessionHistory {
		sessionEnd := session.EventTime
		switch session.Type {
		case Messages.Practice1Session, Messages.Practice2Session, Messages.Practice3Session:
			sessionEnd = sessionEnd.Add(time.Hour * 1)

		case Messages.QualifyingSession:
			sessionEnd = sessionEnd.Add(time.Hour * 1)

		case Messages.SprintSession:
			sessionEnd = sessionEnd.Add(time.Hour * 1)

		case Messages.RaceSession:
			sessionEnd = sessionEnd.Add(time.Hour * 3)
		}

		if sessionEnd.Before(time.Now()) {
			result = append(result, session)
		}
	}

	return result
}

func GetSessionHistory(year int, eventName string, sessionType Messages.SessionType) RaceEvent {
	history := RaceHistory()

	for _, event := range history {
		if event.RaceTime.Year() == year && strings.Contains(event.Name, eventName) && event.Type == sessionType {
			return event
		}
	}
	return RaceEvent{}
}

func HappeningSessions() (liveSession RaceEvent, nextSession RaceEvent, hasLiveSession bool, hasNextSession bool) {
	all := sessionHistory
	utcNow := time.Now().UTC()

	var currentSession *RaceEvent
	var nextUpcomingSession *RaceEvent

	for x := 0; x < len(all); x++ {
		// If we are the same day as a session
		if all[x].EventTime.Year() == utcNow.Year() &&
			all[x].EventTime.Month() == utcNow.Month() &&
			all[x].EventTime.Day() == utcNow.Day() {

			// Check if this session is currently happening
			sessionStart := all[x].EventTime.Add(-time.Minute * 5) // 5 mins before start
			var sessionEnd time.Time

			switch all[x].Type {
			case Messages.Practice1Session, Messages.Practice2Session, Messages.Practice3Session:
				sessionEnd = all[x].EventTime.Add(time.Hour * 2) // 2 hours to cover both 60 and 90 min sessions
			case Messages.QualifyingSession, Messages.SprintSession, Messages.RaceSession, Messages.PreSeasonSession:
				sessionEnd = all[x].EventTime.Add(time.Hour * 4) // 4 hours should cover any race/quali/sprint
			default:
				panic("History: Unhandled session type: " + all[x].Type.String())
			}

			// If we're in the session window
			if utcNow.After(sessionStart) && utcNow.Before(sessionEnd) {
				currentSession = &all[x]
				if x > 0 {
					nextUpcomingSession = &all[x-1]
				}
				break
			}

			// If this session hasn't started yet
			if utcNow.Before(sessionStart) {
				if nextUpcomingSession == nil || nextUpcomingSession.EventTime.After(all[x].EventTime) {
					nextUpcomingSession = &all[x]
				}
			}

			// If nextUpcomingSession is nil here and all sessions for the day are done.
			// We can now return the session before this one as it will be tomorrow.
			if utcNow.After(sessionEnd) && nextUpcomingSession == nil {
				nextUpcomingSession = &all[x-1]
			}

		} else if all[x].EventTime.Before(utcNow) {
			// Past sessions
			if x > 0 && nextUpcomingSession == nil {
				nextUpcomingSession = &all[x-1]
			}
			break
		}
	}

	if currentSession != nil {
		if nextUpcomingSession != nil {
			return *currentSession, *nextUpcomingSession, true, true
		}
		return *currentSession, RaceEvent{}, true, false
	}

	if nextUpcomingSession != nil {
		return RaceEvent{}, *nextUpcomingSession, false, true
	}

	return RaceEvent{}, RaceEvent{}, false, false
}

func liveEvent() (event RaceEvent, exists bool) {
	live, _, exists, _ := HappeningSessions()
	return live, exists
}
