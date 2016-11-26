package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
)

// RFC5424 log message levels.
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

// Legacy loglevel constants to ensure backwards compatibility.
//
// Deprecated: will be removed in 1.5.0.
const (
	LevelInfo  = LevelInformational
	LevelTrace = LevelDebug
	LevelWarn  = LevelWarning
)

type loggerType func() Logger

// Logger defines the behavior of a log provider.
type Logger interface {
	Init(config string) error
	WriteMsg(msg string, level int) error
	Destroy()
	Flush()
}

var adapters = make(map[string]loggerType)

// Register makes a log provide available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, log loggerType) {
	if log == nil {
		panic("logs: Register provide is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("logs: Register called twice for provider " + name)
	}
	adapters[name] = log
}

type GwsLogger struct {
	lock                sync.Mutex
	level               int
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	asynchronous        bool
	msgChan             chan *logMsg
	outputs             []*nameLogger
}

type nameLogger struct {
	Logger
	name string
}

type logMsg struct {
	level int
	msg   string
}

var logMsgPool *sync.Pool

// NewLogger returns a new GwsLogger.
// channelLen means the number of messages in chan(used where asynchronous is true).
// if the buffering chan is full, logger adapters write to file or other way.
func NewLogger(channelLen int64) *GwsLogger {
	bl := new(GwsLogger)
	bl.level = LevelDebug
	bl.loggerFuncCallDepth = 2
	bl.msgChan = make(chan *logMsg, channelLen)
	return bl
}

// Async set the log to asynchronous and start the goroutine
func (bl *GwsLogger) Async() *GwsLogger {
	bl.asynchronous = true
	logMsgPool = &sync.Pool{
		New: func() interface{} {
			return &logMsg{}
		},
	}
	go bl.startLogger()
	return bl
}

// SetLogger provides a given logger adapter into GwsLogger with config string.
// config need to be correct JSON as string: {"interval":360}.
func (bl *GwsLogger) SetLogger(adapterName string, config string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	if log, ok := adapters[adapterName]; ok {
		lg := log()
		err := lg.Init(config)
		if err != nil {
			fmt.Fprintln(os.Stderr, "logs.GwsLogger.SetLogger: "+err.Error())
			return err
		}
		bl.outputs = append(bl.outputs, &nameLogger{name: adapterName, Logger: lg})
	} else {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adapterName)
	}
	return nil
}

// DelLogger remove a logger adapter in GwsLogger.
func (bl *GwsLogger) DelLogger(adapterName string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	outputs := []*nameLogger{}
	for _, lg := range bl.outputs {
		if lg.name == adapterName {
			lg.Destroy()
		} else {
			outputs = append(outputs, lg)
		}
	}
	if len(outputs) == len(bl.outputs) {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adapterName)
	}
	bl.outputs = outputs
	return nil
}

func (bl *GwsLogger) writeToLoggers(msg string, level int) {
	for _, l := range bl.outputs {
		err := l.WriteMsg(msg, level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to WriteMsg to adapter:%v,error:%v\n", l.name, err)
		}
	}
}

func (bl *GwsLogger) writeMsg(logLevel int, msg string) error {
	if bl.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(bl.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		msg = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]" + msg
	}
	if bl.asynchronous {
		lm := logMsgPool.Get().(*logMsg)
		lm.level = logLevel
		lm.msg = msg
		bl.msgChan <- lm
	} else {
		bl.writeToLoggers(msg, logLevel)
	}
	return nil
}

// SetLevel Set log message level.
// If message level (such as LevelDebug) is higher than logger level (such as LevelWarning),
// log providers will not even be sent the message.
func (bl *GwsLogger) SetLevel(l int) {
	bl.level = l
}

// SetLogFuncCallDepth set log funcCallDepth
func (bl *GwsLogger) SetLogFuncCallDepth(d int) {
	bl.loggerFuncCallDepth = d
}

// GetLogFuncCallDepth return log funcCallDepth for wrapper
func (bl *GwsLogger) GetLogFuncCallDepth() int {
	return bl.loggerFuncCallDepth
}

// EnableFuncCallDepth enable log funcCallDepth
func (bl *GwsLogger) EnableFuncCallDepth(b bool) {
	bl.enableFuncCallDepth = b
}

// start logger chan reading.
// when chan is not empty, write logs.
func (bl *GwsLogger) startLogger() {
	for {
		select {
		case bm := <-bl.msgChan:
			bl.writeToLoggers(bm.msg, bm.level)
			logMsgPool.Put(bm)
		}
	}
}

// Emergency Log EMERGENCY level message.
func (bl *GwsLogger) Emergency(format string, v ...interface{}) {
	if LevelEmergency > bl.level {
		return
	}
	msg := fmt.Sprintf("[M] "+format, v...)
	bl.writeMsg(LevelEmergency, msg)
}

// Alert Log ALERT level message.
func (bl *GwsLogger) Alert(format string, v ...interface{}) {
	if LevelAlert > bl.level {
		return
	}
	msg := fmt.Sprintf("[A] "+format, v...)
	bl.writeMsg(LevelAlert, msg)
}

// Critical Log CRITICAL level message.
func (bl *GwsLogger) Critical(format string, v ...interface{}) {
	if LevelCritical > bl.level {
		return
	}
	msg := fmt.Sprintf("[C] "+format, v...)
	bl.writeMsg(LevelCritical, msg)
}

// Error Log ERROR level message.
func (bl *GwsLogger) Error(format string, v ...interface{}) {
	if LevelError > bl.level {
		return
	}
	msg := fmt.Sprintf("[E] "+format, v...)
	bl.writeMsg(LevelError, msg)
}

// Warning Log WARNING level message.
func (bl *GwsLogger) Warning(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writeMsg(LevelWarning, msg)
}

// Notice Log NOTICE level message.
func (bl *GwsLogger) Notice(format string, v ...interface{}) {
	if LevelNotice > bl.level {
		return
	}
	msg := fmt.Sprintf("[N] "+format, v...)
	bl.writeMsg(LevelNotice, msg)
}

// Informational Log INFORMATIONAL level message.
func (bl *GwsLogger) Informational(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writeMsg(LevelInformational, msg)
}

// Debug Log DEBUG level message.
func (bl *GwsLogger) Debug(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writeMsg(LevelDebug, msg)
}

// Warn Log WARN level message.
// compatibility alias for Warning()
func (bl *GwsLogger) Warn(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writeMsg(LevelWarning, msg)
}

// Info Log INFO level message.
// compatibility alias for Informational()
func (bl *GwsLogger) Info(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writeMsg(LevelInformational, msg)
}

// Trace Log TRACE level message.
// compatibility alias for Debug()
func (bl *GwsLogger) Trace(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writeMsg(LevelDebug, msg)
}

// Flush flush all chan data.
func (bl *GwsLogger) Flush() {
	for _, l := range bl.outputs {
		l.Flush()
	}
}

// Close close logger, flush all chan data and destroy all adapters in GwsLogger.
func (bl *GwsLogger) Close() {
	for {
		if len(bl.msgChan) > 0 {
			bm := <-bl.msgChan
			bl.writeToLoggers(bm.msg, bm.level)
			logMsgPool.Put(bm)
			continue
		}
		break
	}
	for _, l := range bl.outputs {
		l.Flush()
		l.Destroy()
	}
}

func (bl *GwsLogger) Printf(format string, v ...interface{}) {
	bl.Info(format,v...)
}

func (bl *GwsLogger) Print(v ...interface{}) {
	bl.writeMsg(LevelInformational, "[I] "+fmt.Sprint(v...))
}

func (bl *GwsLogger) Println(v ...interface{}) {
	bl.Info(fmt.Sprintln(v...))
}