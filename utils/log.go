package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

//log level
const (
	Detail LogLevel = 1
	Debug  LogLevel = 10
	Info   LogLevel = 20
	Warn   LogLevel = 30
	Error  LogLevel = 40
	Fatal  LogLevel = 50
)

type LogLevel int

var level2String = make(map[LogLevel]string)

type Logger struct {
	stdLogger  *log.Logger
	fileLogger *log.Logger
	enablestd  bool
	enablefile bool
	loglevel   LogLevel
}

var MyLogger = NewLogger("test.log", true, true)

func NewLogger(filepath string, enablestd, enablefile bool) *Logger {
	logger := new(Logger)
	logfile := os.Stdout
	logger.stdLogger = log.New(logfile, "\r\n", log.Ldate|log.Ltime|log.Lshortfile)
	logger.stdLogger.SetPrefix("[Info]")

	//init file output
	outfile, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666) //open file, if not exist, create at filepath
	if err != nil {
		fmt.Println("log file open failed:", err.Error())
		os.Exit(1)
	}
	logger.fileLogger = log.New(outfile, "\r\n", log.Ldate|log.Ltime|log.Lshortfile)

	logger.enablefile = enablefile
	logger.enablestd = enablestd
	logger.loglevel = Debug
	return logger

}

func init() {
	level2String[Detail] = "[Detail]"
	level2String[Debug] = "[DEBUG]"
	level2String[Info] = "[INFO]"
	level2String[Warn] = "[WARN]"
	level2String[Error] = "[ERROR]"
	level2String[Fatal] = "[FATAL]"

}

//SetLogLevel
func (l *Logger) SetLogLevel(lv LogLevel) {
	l.loglevel = lv
}

//getLevelString
func (l *Logger) getLevelString(lv LogLevel) string {
	if str, ok := level2String[lv]; ok {
		return str
	} else {

		return "[" + strconv.Itoa(int(lv)) + "]"
	}

}

func (l *Logger) SetEnablestd(b bool) {
	l.enablestd = b
}

func (l *Logger) SetEnablefile(b bool) {
	l.enablefile = b
}

func (l *Logger) Log(level LogLevel, v ...interface{}) {
	if level < l.loglevel {
		return
	}

	if l.enablestd {
		l.stdLogger.SetPrefix(l.getLevelString(level))
		//l.stdLogger.Println(v...)
		l.stdLogger.Output(2, fmt.Sprintln(v...))
	}
	if l.enablefile {
		l.fileLogger.SetPrefix(l.getLevelString(level))
		//l.fileLogger.Println(v...)
		l.fileLogger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) logDepth(level LogLevel, calldepth int, v ...interface{}) {
	if level < l.loglevel {
		return
	}

	if l.enablestd {
		l.stdLogger.SetPrefix(l.getLevelString(level))
		//l.stdLogger.Println(v...)
		l.stdLogger.Output(calldepth, fmt.Sprintln(v...))
	}
	if l.enablefile {
		l.fileLogger.SetPrefix(l.getLevelString(level))
		l.fileLogger.Output(calldepth, fmt.Sprintln(v...))
		//l.fileLogger.Println(v...)
	}
}

func (l *Logger) ErrorLog(v ...interface{}) {
	//l.Log(Error, v...)
	l.logDepth(Error, 3, v...)
}

//Log 调用 默认的日志对象, 输出info日志
func Log(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Info, 3, v...)
}

//ErrorLog 调用 默认的日志对象, 输出error日志
func ErrorLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Error, 3, v...)
}

//DebugLog 调用 默认的日志对象, 输出debug日志
func DebugLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Debug, 3, v...)
}

func CheckError(err error) bool {
	if err != nil {
		MyLogger.ErrorLog("Fatal error:", err.Error())
		return true
	}
	return false
}
