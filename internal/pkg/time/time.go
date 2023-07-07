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

// CalcZeroTime 计算当前时间距离零点的间隔
func CalcZeroTime() time.Duration {
	now := time.Now()
	next := now.Add(time.Hour * 24)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	return next.Sub(now)
}
