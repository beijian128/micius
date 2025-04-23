package util

import (
	"time"
)

const (
	goBornDate      = "2006-01-02"          //长日期格式
	goBornShortDate = "06-01-02"            //短日期格式
	goBornTimes     = "15:04:02"            //长时间格式
	goBornShortTime = "15:04"               //短时间格式
	goBornDateTime  = "2006-01-02 15:04:02" //日期时间格式
)

// 每个月的天数，润年2月天数在具体逻辑中处理
var dayofMonth = [12]int{
	31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31,
}

// GetTodayZeroTime 获取本地时间的零点时间结构体
func GetTodayZeroTime() time.Time {
	timeStr := time.Now().Format(goBornDate)
	zeroTime, _ := time.ParseInLocation(goBornDate, timeStr, time.Now().Location())
	return zeroTime
}

// GetTodayZeroTimeString 获取零点时间字符串 （长日期格式）
func GetTodayZeroTimeString() string {
	timeStr := time.Now().Format(goBornDate)
	return timeStr
}

// GetTodayZeroTimeInt 获取本地时间的零点时间戳，单位秒
func GetTodayZeroTimeInt() int64 {
	return GetTodayZeroTime().Unix()
}

// IntToTime 将时间戳转换为Time结构
func IntToTime(second int64) time.Time {
	return time.Unix(second, 0)
}

// DateStringToInt 将长日期格式的字符串转换为本地时间戳
func DateStringToInt(date string) int64 {
	now, _ := time.ParseInLocation(goBornDate, date, time.Now().Location())
	return now.Unix()
}

// GetTargetDayZeroTime 获取指定时间的零点
func GetTargetDayZeroTime(t time.Time) time.Time {
	timeStr := t.Format(goBornDate)
	zeroTime, _ := time.ParseInLocation(goBornDate, timeStr, time.Now().Location())
	return zeroTime
}

// GetTargetDayZeroTimeInt 获取指定时间零点时间戳
func GetTargetDayZeroTimeInt(t time.Time) int64 {
	return GetTargetDayZeroTime(t).Unix()
}

// GetMonth 获取指定时间所属月份
func GetMonth(t time.Time) int {
	return int(t.Month())
}

// GetWeekDay 获取指定时间所属星期
func GetWeekDay(t time.Time) int {
	return int(t.Weekday())
}

// GetTodayNHourTimeInt 获取本地之间今天第N小时的时间戳
func GetTodayNHourTimeInt(n int) int64 {
	return GetTodayZeroTimeInt() + int64(n)*int64(time.Hour/time.Second)
}

// GetThisWeekFirstDayZeroTime 获取本地时间本周第一天的时间戳(星期一)
func GetThisWeekFirstDayZeroTime() int64 {
	todayZero := GetTodayZeroTime()
	d := GetWeekDay(todayZero)
	if d == 0 {
		d = 7
	}
	return todayZero.Unix() - int64(d-1)*24*60*60
}

// GetTargetDayWeekFirstZeroTimeInt 获取本地指定时间所在周第一天的零点(星期一)
func GetTargetDayWeekFirstZeroTimeInt(t time.Time) int64 {
	todayZero := GetTargetDayZeroTime(t)
	d := GetWeekDay(todayZero)
	if d == 0 {
		d = 7
	}
	return todayZero.Unix() - int64(d-1)*24*60*60
}

// GetThisMonthFirstDayZeroTime 获取本地时间本月第一天的时间戳
func GetThisMonthFirstDayZeroTime() int64 {
	todayZero := GetTodayZeroTime()
	return todayZero.Unix() - int64(todayZero.Day()-1)*24*60*60
}

// GetThisMonthLastDayZeroTime 获取本地时间本月最后一天的时间戳
func GetThisMonthLastDayZeroTime() int64 {
	todayZero := GetTodayZeroTime()
	if todayZero.Month() == time.February && IsLeap(todayZero.Year()) {
		return todayZero.Unix() + int64(dayofMonth[GetMonth(todayZero)-1]-todayZero.Day()+1)*24*60*60
	}
	return todayZero.Unix() + int64(dayofMonth[GetMonth(todayZero)-1]-todayZero.Day())*24*60*60
}

// IsLeap 是否是闰年
func IsLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// IsSameDay 是否同一天
func IsSameDay(left, right int64) bool {
	leftTime := IntToTime(left)
	rightTime := IntToTime(right)
	if GetTargetDayZeroTimeInt(leftTime) == GetTargetDayZeroTimeInt(rightTime) {
		return true
	}
	return false
}

// IsSameWeek 是否同一周 （ 注意：虽然严格来说星期日才是一周的第一天，但这里认为星期一才是第一天。所以是[1,2,3,4,5,6,7]一个周期 ，而不是 [7,1,2,3,4,5,6] ）
func IsSameWeek(left, right int64) bool {
	leftTime := IntToTime(left)
	rightTime := IntToTime(right)
	if GetTargetDayWeekFirstZeroTimeInt(leftTime) == GetTargetDayWeekFirstZeroTimeInt(rightTime) {
		return true
	}
	return false
}

// IsSameMonth 是否同一月
func IsSameMonth(left, right int64) bool {
	time1 := time.Unix(left, 0)
	time2 := time.Unix(right, 0)

	return time1.Year() == time2.Year() && time1.Month() == time2.Month()
}

// DiffDay 获取两个时间戳的相隔天数
func DiffDay(left, right int64) int {
	leftTime := IntToTime(left)
	rightTime := IntToTime(right)

	var diff int64
	if left > right {
		diff = GetTargetDayZeroTimeInt(leftTime) - GetTargetDayZeroTimeInt(rightTime)
	} else {
		diff = GetTargetDayZeroTimeInt(rightTime) - GetTargetDayZeroTimeInt(leftTime)
	}

	return int(diff / (24 * 3600))
}

func SecondsFromTodayZero(t string) int64 {
	// 17:10:23
	// 01234567
	if len(t) != 8 {
		return 0
	}
	h := int64((t[0]-'0')*10 + (t[1] - '0'))
	m := int64((t[3]-'0')*10 + (t[4] - '0'))
	s := int64((t[6]-'0')*10 + (t[7] - '0'))
	return h*3600 + m*60 + s
}
