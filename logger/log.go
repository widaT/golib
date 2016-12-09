package logger

import (
	"sync"
	"fmt"
	"os"
)

const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

// Logger defines the behavior of a log provider.
type Logger interface {
	Init(config string) error
	WriteMsg(msg string) error
	Destroy()
	Flush()
}

type GxLogger struct {
	lock                sync.Mutex
	level               int
	l		   Logger
}
// Init file logger with json config.
// jsonConfig like:
//	{
//	"filename":"logs/beego.log",
//	"maxsize":1<<30,
//	"daily":true,
//	"rotate":true,
//  	"perm":0600
//	}
func NewLogger(jsonConfig string) *GxLogger {
	bl := new(GxLogger)
	bl.level = LevelDebug
	bl.l = newFileWriter()
	bl.l.Init(jsonConfig)
	return bl
}

func (bl *GxLogger) writeToLoggers(msg string) {
	err := bl.l.WriteMsg(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to WriteMsg to File error:%v\n", err)
	}
}

func (bl *GxLogger) writeMsg(msg string) {
	bl.writeToLoggers(msg)
}

func (bl *GxLogger) Flush() {
	bl.l.Flush()
}

func (bl *GxLogger) Close() {
	bl.l.Flush()
	bl.l.Destroy()
}

func (bl *GxLogger) Emergency(format string, v ...interface{}) {
	if LevelEmergency > bl.level {
		return
	}
	msg := fmt.Sprintf("[M] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Alert(format string, v ...interface{}) {
	if LevelAlert > bl.level {
		return
	}
	msg := fmt.Sprintf("[A] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Critical(format string, v ...interface{}) {
	if LevelCritical > bl.level {
		return
	}
	msg := fmt.Sprintf("[C] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Error(format string, v ...interface{}) {
	if LevelError > bl.level {
		return
	}
	msg := fmt.Sprintf("[E] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Warning(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writeMsg( msg)
}

func (bl *GxLogger) Notice(format string, v ...interface{}) {
	if LevelNotice > bl.level {
		return
	}
	msg := fmt.Sprintf("[N] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Informational(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Debug(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Warn(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Info(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writeMsg(msg)
}

func (bl *GxLogger) Trace(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writeMsg(msg)
}
