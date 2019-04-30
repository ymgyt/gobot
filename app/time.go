package app

import "time"

var (
	defaultTimeZone = time.FixedZone("Asia/Tokyo", 9*60*60)
	TimeZone        = defaultTimeZone

	StartTime = Now()
)

func Now() time.Time {
	return time.Now().In(TimeZone)
}
