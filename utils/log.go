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

var MyLogger *Logger // = NewLogger("tmp/logs/stdout.log", true, true)

func newLogger(filepath string, enableStd, enableFile bool) *Logger {
	//init file output
	outfile, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666) //open file, if not exist, create at filepath
	if err != nil {
		panic(fmt.Sprintf("log file '%v' open failed: %v", filepath, err.Error()))
	}

	logger := &Logger{
		stdLogger:  log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile),
		fileLogger: log.New(outfile, "\r\n", log.Ldate|log.Ltime|log.Lshortfile),
		enablefile: enableFile,
		enablestd:  enableStd,
		loglevel:   Debug,
	}

	logger.stdLogger.SetPrefix("[Info]")

	return logger

}

func NewLogger(filepath string, enableStd, enableFile bool) *Logger {
	return newLogger(filepath, enableStd, enableFile)
}

func NewDefaultLogger(filepath string, enableStd, enableFile bool) *Logger {
	MyLogger = newLogger(filepath, enableStd, enableFile)
	return MyLogger
}

func init() {
	level2String[Detail] = "[Detail]"
	level2String[Debug] = "[DEBUG]"
	level2String[Info] = "[INFO]"
	level2String[Warn] = "[WARN]"
	level2String[Error] = "[ERROR]"
	level2String[Fatal] = "[FATAL]"
	MyLogger = newLogger("tmp/logs/stdout.log", true, true)
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

//Log calls default logger and output info log
func Log(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Info, 3, v...)
}

//ErrorLog call default logger and output error log
func ErrorLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Error, 3, v...)
}

//ErrorLogf call default logger and output error log
func ErrorLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Error, 3, fmt.Errorf(template, v...))
}

//DebugLog calls default logger and output debug log
func DebugLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Debug, 3, v...)
}

//DebugLog calls default logger and output debug log
func DebugLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.logDepth(Debug, 3, fmt.Sprintf(template, v...))
}

// CheckError  TODO This is a bad way to call error log, as you cannot know where this method is called in your error log
// This give log line like this : [ERROR]2021/04/13 22:39:11 log.go:150: Fatal error: address 127.0.0.1: missing port in address
// it always refer to this file and this line
// If time allows, a better logging tool like "zerolog" can be used to replace these methods.
func CheckError(err error) bool {
	if err != nil {
		// a fatal error , should be fatal and exit. If that is not the expected behavior, change this log
		MyLogger.ErrorLog("Fatal error:", err.Error())
		return true
	}
	return false
}
