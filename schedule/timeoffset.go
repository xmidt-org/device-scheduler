// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schedule

import "time"

type weeklyOffset int

func ToWeeklyOffset(when time.Time) weeklyOffset {
	return weeklyOffset(
		int(when.Weekday())*24*3600 +
			when.Hour()*3600 +
			when.Minute()*60 +
			when.Second())
}

func (w weeklyOffset) ToTime(relativeTo time.Time) time.Time {
	sun := relativeTo.AddDate(0, 0, -1*int(relativeTo.Weekday()))
	return time.Date(sun.Year(), sun.Month(), sun.Day(), 0, 0, int(w), 0, sun.Location())
}
