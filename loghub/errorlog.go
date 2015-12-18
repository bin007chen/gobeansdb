package loghub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var (
	ErrorLogFormat = "%s %15s:%4d - %s"
	ErrorLogFlag   = (log.LstdFlags | log.Lmicroseconds)
	ErrorLog       *Logger
)

type ErrorLogHub struct {
	logger *log.Logger
	logFd  *os.File
	BufferLog
}

func init() {
	logger := openLogWithFd(os.Stderr, ErrorLogFlag)
	hub := &ErrorLogHub{logger: logger}
	hub.InitBuffer(200)
	ErrorLog = NewLogger("", hub, DEBUG)
}

func InitErrorLog(path string, level int, bufferSize int) (err error) {
	if accesslog, accessFile, err := openLog(path, ErrorLogFlag); err == nil {
		hub := &ErrorLogHub{logger: accesslog}
		hub.InitBuffer(bufferSize)
		hub.logFd = accessFile

		ErrorLog.Hub = hub
		ErrorLog.SetLevel(level)
	} else {
		log.Fatalf("open log error, path=[%s], err=[%s]", path, err.Error())
	}
	return
}

func openLogWithFd(fd *os.File, logFlag int) *log.Logger {
	return log.New(fd, "", logFlag)
}

func openLog(path string, logFlag int) (logger *log.Logger, fd *os.File, err error) {
	if fd, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		logger = openLogWithFd(fd, logFlag)
	}
	return
}

func (hub *ErrorLogHub) Log(name string, level int, file string, line int, msg string) {
	hub.logger.Printf(ErrorLogFormat, levelString[level], file, line, msg)
	bufline := &BufferLine{time.Now(), level, file, line, msg}
	hub.Add(bufline)
	hub.Last[level] = bufline
	if level == FATAL {
		os.Exit(1)
	}
}

func (hub *ErrorLogHub) Reopen(path string) (err error) {
	return nil
}

// Buffer

type BufferLog struct {
	sync.Mutex
	head   int
	Buffer []*BufferLine

	all  queue
	warn queue
	Last [FATAL + 1]*BufferLine
}

func (l *BufferLog) InitBuffer(size int) {
	l.all.Buffer = make([]*BufferLine, size)
	l.warn.Buffer = make([]*BufferLine, size)
}

func (l *BufferLog) DumpBuffer(all bool, out io.Writer) {
	l.Lock()
	defer l.Unlock()
	if all {
		l.all.DumpBuffer(out)
	} else {
		l.warn.DumpBuffer(out)
	}
}

func (l *BufferLog) Add(line *BufferLine) {
	l.Lock()
	defer l.Unlock()
	l.all.Add(line)
	if line.Level >= WARN {
		l.warn.Add(line)
	}
}

func (l *BufferLog) GetLastLog() []byte {
	b, _ := json.Marshal(l.Last[:])
	return b
}

type BufferLine struct {
	TS    time.Time
	Level int
	File  string
	Line  int
	Msg   string
}

type queue struct {
	head   int
	Buffer []*BufferLine
}

func (q *queue) Add(line *BufferLine) {
	q.Buffer[q.head] = line
	q.head += 1
	if q.head >= len(q.Buffer) {
		q.head = 0
	}
}

func (q *queue) DumpBuffer(out io.Writer) {
	i := q.head
	for j := 0; j < len(q.Buffer); j++ {
		line := q.Buffer[i]
		if line != nil {
			out.Write([]byte(fmt.Sprintf("%v\n", line)))
		}
		i += 1
		if i >= len(q.Buffer) {
			i = 0
		}
	}
}
