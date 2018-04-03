package time2

import (
	"time"
	"errors"
)

const DEFAILT_FROMAT = "2006-01-02 15:04:05"
const ONLY_DATE = "2006-01-02"
const ONLY_TIME = "15:04:05"
const SHORT = "2006-1-2 15:4:5"

func TimestampToTime(timestamp int64,param ... string)  (string,error) {
	format := DEFAILT_FROMAT
	if len(param )== 1 {
		format = param[0]
	}else if len(param) > 1 {
		return "",errors.New("wrong param length")
	}
	tm := time.Unix(timestamp, 0)
	return tm.Format(format),nil
}

func TimeToTimestamp(date string ,param ... string) (int64,error) {
	format := DEFAILT_FROMAT
	if len(param )== 1 {
		format = param[0]
	}else if len(param ) > 1{
		return 0,errors.New("wrong param length")
	}
	loc, _ := time.LoadLocation("Local")
	theTime, err := time.ParseInLocation(format, date, loc)
	if err != nil {
		return 0,err
	}
	return theTime.Unix() ,nil
}

func Trans(str string ) string {
	t ,_ := time.Parse(SHORT, str)
	return t.Format(DEFAILT_FROMAT)
}

func OnlyDate( t time.Time) string {
	return t.Format(ONLY_DATE)
}

func OnlyTime( t time.Time) string {
	return t.Format(ONLY_TIME)
}
