package util

import (
	"strconv"
	"time"
)

const MomentLength = 32

func TimeString() string {
	return time.Now().UTC().String()
}

func ParseTimeString(timestr string) time.Time {
	yr, _ := strconv.Atoi(timestr[:4])
	mo, _ := strconv.Atoi(timestr[5:7])
	d, _ := strconv.Atoi(timestr[8:10])
	hr, _ := strconv.Atoi(timestr[11:13])
	min, _ := strconv.Atoi(timestr[14:16])
	sec, _ := strconv.Atoi(timestr[17:19])
	return time.Date(yr, time.Month(mo), d, hr, min, sec, 0, time.Local)
}

var months = map[string]int{
	"Jan": 1,
	"Feb": 2,
	"Mar": 3,
	"Apr": 4,
	"May": 5,
	"Jun": 6,
	"Jul": 7,
	"Aug": 8,
	"Sep": 9,
	"Oct": 10,
	"Nov": 11,
	"Dec": 12,
}

func ParseMomentString(momentstr string) time.Time {
	yr, _ := strconv.Atoi(momentstr[:4])
	mo, _ := months[momentstr[5:7]]
	d, _ := strconv.Atoi(momentstr[8:10])
	hr, _ := strconv.Atoi(momentstr[11:13])
	min, _ := strconv.Atoi(momentstr[14:16])
	return time.Date(yr, time.Month(mo), d, hr, min, 0, 0, time.Local)
}

func ParseDateString(datestr string) time.Time {
	yr, _ := strconv.Atoi(datestr[:4])
	mo, _ := strconv.Atoi(datestr[5:7])
	d, _ := strconv.Atoi(datestr[8:10])
	return time.Date(yr, time.Month(mo), d, 0, 0, 0, 0, time.Local)
}

func ParseMinuteString(minutestr string) time.Time {
	yr, _ := strconv.Atoi(minutestr[:4])
	mo, _ := strconv.Atoi(minutestr[5:7])
	d, _ := strconv.Atoi(minutestr[8:10])
	hr, _ := strconv.Atoi(minutestr[11:13])
	min, _ := strconv.Atoi(minutestr[14:16])
	return time.Date(yr, time.Month(mo), d, hr, min, 0, 0, time.Local)
}

func DurationTimeStrings(timestr1, timestr2 string) time.Duration {
	tm1 := ParseTimeString(timestr1)
	tm2 := ParseTimeString(timestr2)
	return tm2.Sub(tm1)
}

func DurationHours(timestr1, timestr2 string) float64 {
	return DurationTimeStrings(timestr1, timestr2).Hours()
}

func DurationDays(timestr1, timestr2 string) float64 {
	return DurationHours(timestr1, timestr2) / float64(24)
}

func ToTheDay(timestr string) string {
	return timestr[:10]
}

func ToTheHour(timestr string) string {
	return timestr[:13]
}

func ToTheMinute(timestr string) string {
	return timestr[:16]
}

func ToTheSecond(timestr string) string {
	return timestr[:19]
}
