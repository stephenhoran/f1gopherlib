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

package connection

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stephenhoran/f1gopherlib/Messages"
	"github.com/stephenhoran/f1gopherlib/f1log"
)

type fileInfo struct {
	name         string
	data         *bufio.Scanner
	nextLine     string
	nextLineTime time.Time
}

type replay struct {
	log      *f1log.F1GopherLibLog
	cache    string
	dataFeed chan Payload

	eventUrl  string
	session   Messages.SessionType
	eventYear int

	dataFiles []fileInfo

	ctx context.Context
	wg  *sync.WaitGroup

	currentTime     time.Time
	currentTimeLock sync.Mutex

	raceStartTime time.Time
}

const NotFoundResponse = "<?xml version='1.0' encoding='UTF-8'?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>"

func CreateReplay(
	ctx context.Context,
	wg *sync.WaitGroup,
	log *f1log.F1GopherLibLog,
	url string,
	session Messages.SessionType,
	eventYear int,
	cache string) *replay {

	return &replay{
		ctx:       ctx,
		wg:        wg,
		log:       log,
		dataFeed:  make(chan Payload, 1000),
		eventUrl:  url,
		session:   session,
		eventYear: eventYear,
		cache:     cache,
	}
}

func (r *replay) Connect() (error, <-chan Payload) {

	r.dataFiles = make([]fileInfo, 0)

	for _, name := range OrderedFiles {

		if (name == PositionFile || name == ContentStreamsFile) && r.eventYear <= 2018 {
			continue
		}

		if name == LapCountFile && !(r.session == Messages.RaceSession || r.session == Messages.SprintSession) {
			continue
		}

		// Often don't get this data for replays
		if name == AudioStreamsFile {
			continue
		}

		r.dataFiles = append(r.dataFiles, fileInfo{
			name:         name,
			data:         r.get(r.eventUrl + name + ".jsonStream"),
			nextLine:     "",
			nextLineTime: time.Time{},
		})
	}

	go r.readEntries()

	return nil, r.dataFeed
}

func (r *replay) IncrementTime(amount time.Duration) {
	r.currentTimeLock.Lock()
	defer r.currentTimeLock.Unlock()

	r.currentTime = r.currentTime.Add(amount)
}

func (r *replay) JumpToStart() time.Time {
	if r.raceStartTime.IsZero() {
		return time.Time{}
	}

	r.currentTimeLock.Lock()
	defer r.currentTimeLock.Unlock()

	r.currentTime = r.raceStartTime
	// Clear this so we can only do it once and not travel back in time if requested multiple times
	r.raceStartTime = time.Time{}

	return r.currentTime
}

func (r *replay) readEntries() {

	dataStartTime, raceStartTime, err := r.findSessionTimes()
	if err != nil {
		r.dataFeed <- Payload{
			Name: EndOfDataFile,
		}
		return
	}

	r.currentTime = dataStartTime
	r.raceStartTime = raceStartTime

	hasData := true
	r.wg.Add(1)
	defer r.wg.Done()

	// Read drivers list and
	for x := range r.dataFiles {
		if r.dataFiles[x].name == DriverListFile {

			r.dataFiles[x].data.Scan()
			line := r.dataFiles[x].data.Text()

			r.dataFiles[x].nextLineTime, r.dataFiles[x].nextLine, err = r.uncompressedDataTime(line, dataStartTime)
			if err != nil {
				continue
			}

			r.dataFeed <- Payload{
				Name:      r.dataFiles[x].name,
				Data:      []byte(r.dataFiles[x].nextLine),
				Timestamp: r.dataFiles[x].nextLineTime.Format("2006-01-02T15:04:05.999Z"),
			}

			break
		}
	}

	ticker := time.NewTicker(time.Second)
	for hasData {
		select {
		case <-r.ctx.Done():
			ticker.Stop()
			return

		case <-ticker.C:
			r.currentTimeLock.Lock()
			currentTime := r.currentTime
			r.currentTimeLock.Unlock()

			hasData = false

			for x := range r.dataFiles {

				if strings.HasSuffix(r.dataFiles[x].name, ".z") {
					hasData = r.sim(
						r.dataFiles[x].data,
						currentTime,
						&r.dataFiles[x].nextLineTime,
						&r.dataFiles[x].nextLine,
						dataStartTime,
						r.compressedDataTime,
						r.dataFiles[x].name) || hasData
				} else {
					hasData = r.sim(
						r.dataFiles[x].data,
						currentTime,
						&r.dataFiles[x].nextLineTime,
						&r.dataFiles[x].nextLine,
						dataStartTime,
						r.uncompressedDataTime,
						r.dataFiles[x].name) || hasData
				}
			}

			currentTime = currentTime.Add(time.Second)
			r.currentTimeLock.Lock()
			// The user can increment the time independantly of us so check we are actually incrementing
			if currentTime.After(r.currentTime) {
				r.currentTime = currentTime
			}
			r.currentTimeLock.Unlock()

		default:
		}
	}

	r.dataFeed <- Payload{
		Name: EndOfDataFile,
	}
}

func (r *replay) sim(
	dataBuffer *bufio.Scanner,
	currentRaceTime time.Time,
	nextTime *time.Time,
	nextData *string,
	sessionStartTime time.Time,
	splitData func(data string, sessionStart time.Time) (timestamp time.Time, payload string, err error),
	name string) bool {

	// If no data then skip
	if dataBuffer == nil {
		return false
	}

	if *nextData != "" {
		if nextTime.After(currentRaceTime) {
			return true
		}

		r.dataFeed <- Payload{
			Name:      name,
			Data:      []byte(*nextData),
			Timestamp: nextTime.Format("2006-01-02T15:04:05.999Z"),
		}
	}

	var err error
	for dataBuffer.Scan() {
		line := dataBuffer.Text()

		if line == NotFoundResponse {
			r.log.Errorf("Replay file not found '%s'", name)
			return false
		}

		*nextTime, *nextData, err = splitData(line, sessionStartTime)
		if err != nil {
			continue
		}

		if nextTime.After(currentRaceTime) {
			return true
		}

		r.dataFeed <- Payload{
			Name:      name,
			Data:      []byte(*nextData),
			Timestamp: nextTime.Format("2006-01-02T15:04:05.999Z"),
		}
	}

	if !dataBuffer.Scan() {
		*nextData = ""
		return false
	}

	return true
}

func (r *replay) findSessionTimes() (dataStartTime time.Time, sessionStartTime time.Time, err error) {
	dataBuffer := r.get(r.eventUrl + ExtrapolatedClockFile + ".jsonStream")

	if dataBuffer == nil {
		r.log.Errorf("Unable to find session start time because file doesn't exist")
		return time.Time{}, time.Time{}, errors.New("No file for session start time")
	}

	dataBuffer.Scan()
	line := dataBuffer.Text()

	if line == NotFoundResponse {
		r.log.Errorf("Session start time not found because %s file was not found.", ExtrapolatedClockFile)
		return time.Time{}, time.Time{}, errors.New("session start time not found")
	}

	var offset time.Duration
	dataStartTime, offset, err = r.timeFromSessionData(line)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	dataBuffer.Scan()
	line = dataBuffer.Text()
	sessionStartTime, _, err = r.timeFromSessionData(line)

	// For session start go back 10 seconds so the UI has chance to redraw before the
	// race starts
	return dataStartTime.Add(-offset), sessionStartTime.Add(-time.Second * 10), err
}

func (r *replay) timeFromSessionData(line string) (currentTime time.Time, offsetFromStart time.Duration, err error) {
	timeEnd := strings.Index(line, "{")
	data := line[timeEnd:]
	timestamp := line[timeEnd-12 : timeEnd]

	abc := fmt.Sprintf("%sh%sm%ss%sms", timestamp[:2], timestamp[3:5], timestamp[6:8], timestamp[9:12])

	offsetFromStart, err = time.ParseDuration(abc)

	var dat map[string]interface{}
	if err := json.Unmarshal([]byte(data), &dat); err != nil {
		r.log.Errorf("Session start date file was invalid: %s", err)
		return time.Time{}, 0, err
	}

	timestampStr := dat["Utc"].(string)

	sessionUtc, err := time.Parse("2006-01-02T15:04:05.9999999Z", timestampStr)
	if err != nil {
		r.log.Errorf("Session start timestamp was invalid: %s", err)
		return time.Time{}, 0, err
	}

	return sessionUtc, offsetFromStart, nil
}

func (r *replay) uncompressedDataTime(data string, sessionStart time.Time) (timestamp time.Time, payload string, err error) {
	timeEnd := strings.Index(data, "{")

	timestamp, err = r.raceTime(data[timeEnd-12:timeEnd], sessionStart)
	if err != nil {
		return time.Time{}, "", err
	}

	return timestamp, data[timeEnd:], nil
}

func (r *replay) compressedDataTime(data string, sessionStart time.Time) (timestamp time.Time, payload string, err error) {
	timeEnd := strings.Index(data, "\"")

	timestamp, err = r.raceTime(data[timeEnd-12:timeEnd], sessionStart)
	if err != nil {
		return time.Time{}, "", err
	}

	return timestamp, data[timeEnd+1 : len(data)-1], nil
}

func (r *replay) raceTime(value string, sessionStart time.Time) (time.Time, error) {

	// There is some weird characters at the start of the string you can't see
	abc := fmt.Sprintf("%sh%sm%ss%sms", value[:2], value[3:5], value[6:8], value[9:12])

	timestamp, err := time.ParseDuration(abc)
	if err != nil {
		r.log.Errorf("Replay error parsing time '%s': %s", abc, err)
		return time.Time{}, err
	}

	return sessionStart.Add(timestamp), nil
}

func (r *replay) get(url string) *bufio.Scanner {

	if len(r.cache) > 0 {
		fileName := filepath.Base(url)

		// If file matching url doesn't exist then retrieve
		cachedFile := filepath.Join(r.cache, fileName)
		cachedFile, _ = filepath.Abs(cachedFile)
		f, err := os.Open(cachedFile)

		if os.IsNotExist(err) {
			f.Close()

			var resp *http.Response
			resp, err = http.Get(url)
			if err != nil {
				r.log.Errorf("Replay url error for '%s': %s", url, err)
				return nil
			}
			defer resp.Body.Close()

			if resp.ContentLength == int64(len(NotFoundResponse)) {
				content, _ := io.ReadAll(resp.Body)
				if string(content) == NotFoundResponse {
					r.log.Errorf("Replay url not found '%s'", url)
					return nil
				}
			}

			scanner := bufio.NewScanner(resp.Body)

			err = os.MkdirAll(filepath.Dir(cachedFile), 0755)

			// Write body to file - using url as name
			var newFile *os.File
			newFile, err = os.Create(cachedFile)
			defer newFile.Close()
			for scanner.Scan() {
				_, err = newFile.Write(scanner.Bytes())

				// need newline for scanner to split
				newFile.WriteString("\n")
			}
			f, err = os.Open(cachedFile)
		}

		return bufio.NewScanner(f)
	}

	var resp *http.Response
	resp, err := http.Get(url)
	if err != nil {
		r.log.Errorf("Replay get url '%s': %s", err)
		return nil
	}
	// TODO - probably need to tidy this up but if we have no cache then we can't close it here or no data
	//defer resp.Body.Close()

	if resp.ContentLength == int64(len(NotFoundResponse)) {
		content, _ := io.ReadAll(resp.Body)
		if string(content) == NotFoundResponse {
			r.log.Errorf("Replay url not found '%s'", url)
			return nil
		}
	}

	return bufio.NewScanner(resp.Body)
}
