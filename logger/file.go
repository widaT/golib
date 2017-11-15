// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// fileLogWriter implements LoggerInterface.
// It writes messages by lines limit, file size limit, or time frequency.
type fileLogWriter struct {
	sync.Mutex // write log order by order and  atomic incr maxLinesCurLines and maxSizeCurSize
	// The opened file
	Filename   string `json:"filename"`
	fileWriter *os.File
	// Rotate at size
	MaxSize        int `json:"maxsize"`
	maxSizeCurSize int
	// Rotate daily
	Daily         bool  `json:"daily"`
	MaxDays       int64 `json:"maxdays"`
	dailyOpenDate int
	Rotate bool `json:"rotate"`
	Perm os.FileMode `json:"perm"`
}

// NewFileWriter create a FileLogWriter returning as LoggerInterface.
func newFileWriter() Logger {
	w := &fileLogWriter{
		Filename: "",
		MaxSize:  1 << 30, //1G
		Daily:    true,
		Rotate:   true,
		Perm:     0660,
	}
	return w
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
func (w *fileLogWriter) Init(jsonConfig string) error {
	err := json.Unmarshal([]byte(jsonConfig), w)
	if err != nil {
		return err
	}
	if len(w.Filename) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	err = w.startLogger()
	return err
}

// start file logger. create log file and set to locker-inside file writer.
func (w *fileLogWriter) startLogger() error {
	file, err := w.createLogFile()
	if err != nil {
		return err
	}
	if w.fileWriter != nil {
		w.fileWriter.Close()
	}
	w.fileWriter = file
	return w.initFd()
}

func (w *fileLogWriter) needRotate(size int, day int) bool {
	return (w.MaxSize > 0 && w.maxSizeCurSize >= w.MaxSize) ||
		(w.Daily && day != w.dailyOpenDate)

}

// WriteMsg write logger message into file.
func (w *fileLogWriter) WriteMsg(msg string) error {
	//2016/01/12 21:34:33
	now := time.Now()
	y, mo, d := now.Date()
	h, mi, s := now.Clock()
	//len(2006/01/02 15:03:04)==19
	var buf [20]byte
	t := 3
	for y >= 10 {
		p := y / 10
		buf[t] = byte('0' + y - p*10)
		y = p
		t--
	}
	buf[0] = byte('0' + y)
	buf[4] = '/'
	if mo > 9 {
		buf[5] = '1'
		buf[6] = byte('0' + mo - 10)
	} else {
		buf[5] = '0'
		buf[6] = byte('0' + mo)
	}
	buf[7] = '/'
	t = d / 10
	buf[8] = byte('0' + t)
	buf[9] = byte('0' + d - t*10)
	buf[10] = ' '
	t = h / 10
	buf[11] = byte('0' + t)
	buf[12] = byte('0' + h - t*10)
	buf[13] = ':'
	t = mi / 10
	buf[14] = byte('0' + t)
	buf[15] = byte('0' + mi - t*10)
	buf[16] = ':'
	t = s / 10
	buf[17] = byte('0' + t)
	buf[18] = byte('0' + s - t*10)
	buf[19] = '\t'
	msg = string(buf[0:]) + msg + "\n"

	if w.Rotate {
		if w.needRotate(len(msg), d) {
			w.Lock()
			if w.needRotate(len(msg), d) {
				if err := w.doRotate(); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
				}

			}
			w.Unlock()
		}
	}
	w.Lock()
	_, err := w.fileWriter.Write([]byte(msg))
	if err == nil {
		w.maxSizeCurSize += len(msg)
	}
	w.Unlock()
	return err
}

func (w *fileLogWriter) createLogFile() (*os.File, error) {
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, w.Perm)
	return fd, err
}

func (w *fileLogWriter) initFd() error {
	fd := w.fileWriter
	fInfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s\n", err)
	}
	w.maxSizeCurSize = int(fInfo.Size())
	w.dailyOpenDate = time.Now().Day()
	return nil
}
func (w *fileLogWriter) doRotate() error {
	_, err := os.Lstat(w.Filename)
	if err != nil {
		return err
	}

	fName := ""
	suffix := filepath.Ext(w.Filename)
	filenameOnly := strings.TrimSuffix(w.Filename, suffix)
	if suffix == "" {
		suffix = ".log"
	}
	fName = filenameOnly + fmt.Sprintf(".%s%s", time.Now().AddDate(0,0,-1).Format("2006-01-02"), suffix)
	_, err = os.Lstat(fName)

	if err == nil {
		return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.Filename)
	}

	// close fileWriter before rename
	w.fileWriter.Close()

	// Rename the file to its new found name
	// even if occurs error,we MUST guarantee to  restart new logger
	renameErr := os.Rename(w.Filename, fName)
	// re-start logger
	startLoggerErr := w.startLogger()
	if startLoggerErr != nil {
		return fmt.Errorf("Rotate StartLogger: %s\n", startLoggerErr)
	}
	if renameErr != nil {
		return fmt.Errorf("Rotate: %s\n", renameErr)
	}
	return nil
}

// Destroy close the file description, close file writer.
func (w *fileLogWriter) Destroy() {
	w.fileWriter.Close()
}

// Flush flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (w *fileLogWriter) Flush() {
	w.fileWriter.Sync()
}