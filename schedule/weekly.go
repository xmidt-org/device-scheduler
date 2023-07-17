// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"fmt"
	"sort"
	"time"
)

type weeklyEntry struct {
	Indexes []int        `codec:"indexes"`
	Time    weeklyOffset `codec:"time"`

	macs []string `codec:"-"`
}

func (entry *weeklyEntry) Finalize(macs []string) error {
	entry.macs = make([]string, len(entry.Indexes))
	for i, idx := range entry.Indexes {
		if idx < 0 || idx > len(macs) {
			return fmt.Errorf("%w: 'indexes' value out of bounds", ErrInvalidInput)
		}
		entry.macs[i] = macs[idx]
	}

	return nil
}

type weeklyList []weeklyEntry

func (p *weeklyList) Finalize(macs []string) error {
	list := *p
	sort.Slice(list, func(i, j int) bool {
		return list[i].Time < list[j].Time
	})

	for i := range list {
		err := list[i].Finalize(macs)
		if err != nil {
			return err
		}
	}

	// Duplicate the last entry of the week to be negative time so it happens
	// the prior week & can seed the current week with the correct state.  Also
	// duplicate the first entry of the week and add a week of time to it so the
	// first change next week is easy to calculate.
	if len(list) > 0 {
		nextWeek := list[0]
		nextWeek.Time += oneWeek

		priorWeek := list[len(list)-1]
		priorWeek.Time -= oneWeek
		list = append([]weeklyEntry{priorWeek}, list...)
		*p = append(list, nextWeek)
	}

	return nil
}

func (list weeklyList) findIndex(when time.Time) int {
	offset := ToWeeklyOffset(when)

	var index int
	for i, entry := range list {
		if entry.Time <= offset {
			index = i
		}
	}
	return index
}

func (list weeklyList) Blocked(when time.Time) (at time.Time, blocked []string) {
	if len(list) == 0 {
		return time.Time{}, nil
	}

	index := list.findIndex(when)

	at = list[index].Time.ToTime(when)
	blocked = list[index].macs

	return at, blocked
}

func (list weeklyList) Until(when time.Time) time.Time {
	if len(list) == 0 {
		return time.Time{}
	}

	index := list.findIndex(when)

	// Look at the next entry
	index++

	fmt.Printf("index: %d [%d]\n", index, list[index].Time)
	return list[index].Time.ToTime(when)
}
