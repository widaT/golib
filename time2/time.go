package time2

import (
	"time"
	"github.com/iris-contrib/errors"
)

const DEFAILT_FROMAT = "2006-01-02 15:04:05"
const ONLY_DATE = "2006-01-02"
const ONLY_TIME = "15:04:05"

func TimestampToTime(timestamp int64,param ... string)  (string,error) {
	format := DEFAILT_FROMAT
	if len(param == 1) {
		format = param[0]
	}else if (len(param > 1)){
		return 0,errors.New("wrong param length")
	}
	tm := time.Unix(timestamp, 0)
	return tm.Format(format),nil
}

func TimeToTimestamp(date string ,param ... string) (int64,error) {
	format := DEFAILT_FROMAT
	if len(param == 1) {
		format = param[0]
	}else if (len(param > 1)){
		return 0,errors.New("wrong param length")
	}
	tm, err := time.Parse(format, date)
	if err != nil {
		return 0,err
	}
	return tm.Unix() ,nil
}
