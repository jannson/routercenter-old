package rcenter

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	colors map[string]string
	logNo  uint64
)

const (
	Black = (iota + 30)
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type Worker struct {
	Minion  *log.Logger
	Color   int
	LogFile *os.File
}

type Info struct {
	Id      uint64
	Time    string
	Module  string
	Level   string
	Message string
	format  string
}

type ServerLogger struct {
	Module string
	Worker *Worker
}

func (info *Info) Output() string {
	msg := fmt.Sprintf(info.format, info.Id, info.Time, info.Level, info.Message)
	return msg
}

func NewWorker(prefix string, flag int, color int, out io.Writer) *Worker {
	return &Worker{Minion: log.New(out, prefix, flag), Color: color, LogFile: nil}
}

func NewConsoleWorker(prefix string, flag int, color int) *Worker {
	return NewWorker(prefix, flag, color, os.Stdout)
}

func NewFileWorker(prefix string, flag int, color int, logFile *os.File) *Worker {
	return &Worker{Minion: log.New(logFile, prefix, flag), Color: color, LogFile: logFile}
}

func (w *Worker) Log(level string, calldepth int, info *Info) error {
	if w.Color != 0 {
		buf := &bytes.Buffer{}
		buf.Write([]byte(colors[level]))
		buf.Write([]byte(info.Output()))
		buf.Write([]byte("\033[0m"))
		return w.Minion.Output(calldepth+1, buf.String())
	} else {
		return w.Minion.Output(calldepth+1, info.Output())
	}
}

func colorString(color int) string {
	return fmt.Sprintf("\033[%dm", color)
}

func initColors() {
	colors = map[string]string{
		"CRITICAL": colorString(Magenta),
		"ERROR":    colorString(Red),
		"WARNING":  colorString(Yellow),
		"NOTICE":   colorString(Green),
		"DEBUG":    colorString(Cyan),
		"INFO":     colorString(White),
	}
}

func NewLogger(module string, color int) (*ServerLogger, error) {
	initColors()
	newWorker := NewConsoleWorker("", 0, color)
	return &ServerLogger{Module: module, Worker: newWorker}, nil
}

func NewFileLogger(module string, color int, logFile string) (*ServerLogger, error) {
	fileHandler, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	} else {
		initColors()
		newWorker := NewFileWorker("", 0, color, fileHandler)
		return &ServerLogger{Module: module, Worker: newWorker}, nil
	}
}

func NewDailyLogger(module string, color int, logPath string) (*ServerLogger, error) {
	var logFile string
	const layout = "2006-01-02"
	now := time.Now()
	fileName := now.Format(layout)
	if len(logPath) == 0 {
		logFile = "./" + fileName + ".log"
	} else {
		logFile = logPath + "/" + fileName + ".log"
	}
	fileHandler, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	} else {
		initColors()
		newWorker := NewFileWorker("", 0, color, fileHandler)
		return &ServerLogger{Module: module, Worker: newWorker}, nil
	}
}

func (logger *ServerLogger) Log(level string, message string) {
	var formatString string = "#%d %s > %.3s %s"
	info := &Info{
		Id:      atomic.AddUint64(&logNo, 1),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Module:  logger.Module,
		Level:   level,
		Message: message,
		format:  formatString,
	}
	logger.Worker.Log(level, 2, info)
}

func (logger *ServerLogger) Fatal(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("CRITICAL", message)
	os.Exit(1)
}

func (logger *ServerLogger) Panic(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("CRITICAL", message)
	panic(message)
}

func (logger *ServerLogger) Critical(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("CRITICAL", message)
}

func (logger *ServerLogger) Error(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("ERROR", message)
}

func (logger *ServerLogger) Warning(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("WARNING", message)
}

func (logger *ServerLogger) Notice(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("NOTICE", message)
}

func (logger *ServerLogger) Info(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("INFO", message)
}

func (logger *ServerLogger) Debug(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	logger.Log("DEBUG", message)
}

func (logger *ServerLogger) Strack(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	message += "\n"
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	message += string(buf[:n])
	message += "\n"
	logger.Log("STRACK", message)
}

func (logger *ServerLogger) Close() {
	if logger.Worker.LogFile != nil {
		logger.Worker.LogFile.Close()
	}
}
