package time

import "time"

const (
	msInSecond      int64 = 1e3
	nsInMillisecond int64 = 1e6
)

// UnixToMS Converts Unix Epoch from milliseconds to time.Time
func UnixToMS(ms int64) time.Time {
	return time.Unix(ms/msInSecond, (ms%msInSecond)*nsInMillisecond)
}

func GetDateTimeStr(time2 time.Time) string {
	return time2.Format("2006-01-02 15:04:05.999")
}

func UtilNextMillis(lastTimestamp int64) int64 {
	var ts = time.Now().UnixMilli()
	for ts <= lastTimestamp {
		ts = time.Now().UnixMilli()
	}
	return ts
}
