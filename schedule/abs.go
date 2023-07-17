// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"fmt"
	"sort"
	"time"
)

type absEntry struct {
	Indexes  []int `codec:"indexes"`
	UnixTime int64 `codec:"unix_time"`

	macs []string  `codec:"-"`
	time time.Time `codec:"-"`
}

func (entry *absEntry) Finalize(macs []string, tz *time.Location) error {
	entry.macs = make([]string, len(entry.Indexes))
	for i, idx := range entry.Indexes {
		if idx < 0 || idx > len(macs) {
			return fmt.Errorf("%w: 'indexes' value out of bounds", ErrInvalidInput)
		}
		entry.macs[i] = macs[idx]
		entry.time = time.Unix(entry.UnixTime, 0).In(tz)
	}

	return nil
}

type absList []absEntry

func (list absList) Finalize(macs []string, tz *time.Location) error {
	sort.Slice(list, func(i, j int) bool {
		return list[i].UnixTime < list[j].UnixTime
	})

	for i := range list {
		err := list[i].Finalize(macs, tz)
		if err != nil {
			return err
		}
	}

	return nil
}

func (list absList) findIndex(when time.Time) int {
	var at time.Time
	index := -1

	for i, entry := range list {
		// !Before == After or Equal
		if entry.time.After(at) && !when.Before(entry.time) {
			at = entry.time
			index = i
		}
	}
	return index
}

func (list absList) Blocked(when time.Time) (at time.Time, blocked []string, definitive bool) {
	index := list.findIndex(when)

	if index > -1 {
		at = list[index].time
		blocked = list[index].macs
		if index < len(list)-1 {
			definitive = true
		}
	}

	return at, blocked, definitive
}

func (list absList) Until(when time.Time) time.Time {
	if len(list) == 0 {
		return time.Time{}
	}

	index := list.findIndex(when)

	// Look at the next entry
	index++
	if len(list) == index {
		// That was the last absolute entry, so return time 0.
		return time.Time{}
	}

	return list[index].time
}
