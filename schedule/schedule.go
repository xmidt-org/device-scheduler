// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ugorji/go/codec"
)

const (
	oneWeek = 7 * 24 * 60 * 60
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

type Schedule struct {
	handle codec.Handle
	raw    schedule
	m      sync.Mutex
}

func New() *Schedule {
	return &Schedule{
		handle: new(codec.MsgpackHandle),
	}
}

func (s *Schedule) Decode(in []byte) error {
	s.m.Lock()
	defer s.m.Unlock()

	var raw schedule
	err := codec.NewDecoderBytes(in, s.handle).Decode(&raw)
	if err != nil {
		return err
	}

	err = raw.Finalize()
	if err != nil {
		return err
	}
	s.raw = raw

	return nil
}

func (s *Schedule) Blocked(when time.Time) []string {
	s.m.Lock()
	defer s.m.Unlock()

	absAt, absBlocked, definitive := s.raw.Absolute.Blocked(when)
	if definitive {
		return absBlocked
	}

	weeklyAt, weeklyBlocked := s.raw.Weekly.Blocked(when)
	if weeklyAt.Before(absAt) {
		return absBlocked
	}

	return weeklyBlocked
}

func (s *Schedule) Until(when time.Time) time.Time {
	s.m.Lock()
	defer s.m.Unlock()

	next := s.raw.Absolute.Until(when)
	if !next.IsZero() {
		return next
	}
	return s.raw.Weekly.Until(when)
}

type schedule struct {
	TimeZone   string     `codec:"time_zone"`
	ReportRate int        `codec:"report_rate"`
	MACs       []string   `codec:"macs"`
	Absolute   absList    `codec:"absolute"`
	Weekly     weeklyList `codec:"weekly"`

	tz         *time.Location `codec:"-"`
	reportRate time.Duration  `codec:"-"`
}

func (s *schedule) Finalize() error {
	tz, err := time.LoadLocation(s.TimeZone)
	if err != nil {
		return fmt.Errorf("%w: 'time_zone' of '%s' invalid: %v", ErrInvalidInput, s.TimeZone, err)
	}
	s.tz = tz

	if s.ReportRate < 0 {
		return fmt.Errorf("%w: 'report_rate' must be 0 or larger", ErrInvalidInput)
	}
	s.reportRate = time.Second * time.Duration(s.ReportRate)

	if err := s.Absolute.Finalize(s.MACs, s.tz); err != nil {
		return err
	}
	if err := s.Weekly.Finalize(s.MACs); err != nil {
		return err
	}

	return nil
}

func (s schedule) String() string {
	var buf strings.Builder

	fmt.Fprintln(&buf, "schedule {")
	fmt.Fprintln(&buf, "\ttime_zone:")
	fmt.Fprintf(&buf, "\t\trequested: %q\n", s.TimeZone)
	fmt.Fprintf(&buf, "\t\tactual:    %q\n", s.tz)
	fmt.Fprintf(&buf, "\treport_rate: %s\n", s.reportRate)
	fmt.Fprintf(&buf, "\tmac_count:   %d\n", len(s.MACs))
	for i, mac := range s.MACs {
		fmt.Fprintf(&buf, "\t\t[%d]: %q\n", i, mac)
	}

	fmt.Fprintln(&buf, "\tabsolute:")
	if len(s.Absolute) == 0 {
		fmt.Fprintln(&buf, "\t\tnone")
	} else {
		for _, entry := range s.Absolute {
			fmt.Fprintf(&buf, "\t\ttime: %d (%s), block: [%s]\n", entry.time.Unix(), entry.time.Format(time.RFC3339), strings.Join(entry.macs, ", "))
		}
	}

	fmt.Fprintln(&buf, "\tweekly:")
	if len(s.Weekly) == 0 {
		fmt.Fprintln(&buf, "\t\tnone")
	} else {
		for _, entry := range s.Weekly {
			switch {
			case entry.Time < 0:
				fmt.Fprintf(&buf, "\t\ttime: %d-%d, block: [%s]\n", entry.Time+oneWeek, oneWeek, strings.Join(entry.macs, ", "))
			case entry.Time >= oneWeek:
				fmt.Fprintf(&buf, "\t\ttime: %d+%d, block: [%s]\n", entry.Time-oneWeek, oneWeek, strings.Join(entry.macs, ", "))
			default:
				fmt.Fprintf(&buf, "\t\ttime: %d, block: [%s]\n", entry.Time, strings.Join(entry.macs, ", "))
			}
		}
	}
	fmt.Fprintln(&buf, "}")

	return buf.String()
}
