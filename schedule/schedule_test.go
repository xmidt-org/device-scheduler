// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		description string
		in          string
	}{
		{
			description: "basic test",
			in: "\x84" +
				"" + "\xa6" + "weekly" + /* : */
				"" + "" + "\x93" + /* [3] */
				"" + "" + "" + "\x82" +
				"" + "" + "" + "" + "\xa4" + "time" + /* : */ "\x0a" +
				"" + "" + "" + "" + "\xa7" + "indexes" + /* : */
				"" + "" + "" + "" + "" + "\x93" + /* [3] */
				"" + "" + "" + "" + "" + "" + "\x00" +
				"" + "" + "" + "" + "" + "" + "\x01" +
				"" + "" + "" + "" + "" + "" + "\x03" +
				"" + "" + "" + "\x82" +
				"" + "" + "" + "" + "\xa4" + "time" + /* : */ "\x14" +
				"" + "" + "" + "" + "\xa7" + "indexes" + /* : */
				"" + "" + "" + "" + "" + "\x91" + /* [1] */
				"" + "" + "" + "" + "" + "" + "\x00" +
				"" + "" + "" + "\x82" +
				"" + "" + "" + "" + "\xa4" + "time" + /* : */ "\x1e" +
				"" + "" + "" + "" + "\xa7" + "indexes" + /* : */
				"" + "" + "" + "" + "" + "\xc0" + /* none */
				"" + "\xa4" + "macs" + /* : */
				"" + "" + "\x94" + /* [4] */
				"" + "" + "" + "\xb1" + "11:22:33:44:55:aa" +
				"" + "" + "" + "\xb1" + "22:33:44:55:66:bb" +
				"" + "" + "" + "\xb1" + "33:44:55:66:77:cc" +
				"" + "" + "" + "\xb1" + "44:55:66:77:88:dd" +
				"" + "\xa8" + "absolute" + /* : */
				"" + "" + "\x91" + /* [1] */
				"" + "" + "" + "\x82" +
				"" + "" + "" + "" + "\xa9" + "unix_time" + /* : */ "\xce" + "\x59\xe5\x83\x17" +
				"" + "" + "" + "" + "\xa7" + "indexes" + /* : */
				"" + "" + "" + "" + "" + "\x92" + /* [2] */
				"" + "" + "" + "" + "" + "" + "\x00" +
				"" + "" + "" + "" + "" + "" + "\x02" +
				"" + "\xab" + "report_rate" + /* : */ "\xce" + "\x00\x01\x51\x80" +
				"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			s := New()
			err := s.Decode([]byte(tc.in))
			assert.NoError(err)
			fmt.Println(s.raw)
		})
	}
}

func TestSimple(t *testing.T) {
	simple := schedule{
		TimeZone:   "UTC",
		ReportRate: 3600,
		MACs:       []string{"11:22:33:44:55:66", "22:33:44:55:66:aa", "33:44:55:66:aa:BB"},
		Absolute: absList{
			{UnixTime: 864010, Indexes: []int{2, 1}},
			{UnixTime: 864015, Indexes: []int{2}},
		},
		Weekly: weeklyList{
			{Time: 23, Indexes: []int{0}},
			{Time: 24, Indexes: []int{1}},
		},
	}

	type check struct {
		UnixTime int64
		MACs     []string
		NextTime int64
	}

	tests := []struct {
		description string
		in          schedule
		checks      []check
	}{
		{
			description: "simple tests from original c code",
			in:          simple,
			checks: []check{
				{UnixTime: 864000, NextTime: 864010, MACs: []string{"22:33:44:55:66:aa"}},
				{UnixTime: 864009, NextTime: 864010, MACs: []string{"22:33:44:55:66:aa"}},
				{UnixTime: 864010, NextTime: 864015, MACs: []string{"33:44:55:66:aa:BB", "22:33:44:55:66:aa"}},
				{UnixTime: 864011, NextTime: 864015, MACs: []string{"33:44:55:66:aa:BB", "22:33:44:55:66:aa"}},
				{UnixTime: 864014, NextTime: 864015, MACs: []string{"33:44:55:66:aa:BB", "22:33:44:55:66:aa"}},
				{UnixTime: 864015, NextTime: 864023, MACs: []string{"33:44:55:66:aa:BB"}},
				{UnixTime: 864016, NextTime: 864023, MACs: []string{"33:44:55:66:aa:BB"}},
				{UnixTime: 864023, NextTime: 864024, MACs: []string{"11:22:33:44:55:66"}},
				{UnixTime: 864024, NextTime: 1468823, MACs: []string{"22:33:44:55:66:aa"}},
				{UnixTime: 864025, NextTime: 1468823, MACs: []string{"22:33:44:55:66:aa"}},
			},
		},
		{
			description: "an empty schedule",
			checks: []check{
				{UnixTime: 864000},
			},
		},
	}

	for _, tc := range tests {
		for _, c := range tc.checks {
			desc := fmt.Sprintf("%s: %d\n", tc.description, c.UnixTime)
			t.Run(desc, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				tmp := tc.in
				require.NoError(tmp.Finalize())

				s := New()

				s.raw = tmp

				fmt.Println(tmp)

				var when time.Time
				if c.UnixTime != 0 {
					when = time.Unix(c.UnixTime, 0).UTC()
				}

				macs := s.Blocked(when)
				assert.Equal(c.MACs, macs)

				var expected time.Time
				if c.NextTime != 0 {
					expected = time.Unix(c.NextTime, 0).UTC()
				}
				got := s.Until(when)
				assert.Equal(expected, got)
			})
		}
	}
}
