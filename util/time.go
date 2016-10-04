package util

import (
	"strconv"
	"time"
)

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
	return time.Date(yr, time.Month(mo), d, hr, min, sec, 0, time.UTC)
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
