package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// log level
const (
	Detail LogLevel = 1
	Debug  LogLevel = 10
	Info   LogLevel = 20
	Warn   LogLevel = 30
	Error  LogLevel = 40
	Fatal  LogLevel = 50
)

var (
	level2String = make(map[LogLevel]string)
	TxLogger     *CombinedLogger
)

type LogLevel int

type Logger struct {
	logger   *log.Logger
	enabled  bool
	logLevel LogLevel
}

type CombinedLogger struct {
	stdLogger  *Logger
	fileLogger *Logger
}

func newLogger(logFilepath string, enableStd, enableFile bool) *CombinedLogger {
	var outfile *os.File
	if enableFile {
		var err error
		if err = os.MkdirAll(filepath.Dir(logFilepath), os.ModePerm); err != nil {
			panic(fmt.Sprintf("log file '%v' initialize failed: %v", logFilepath, err.Error()))
		}
		//init file output
		outfile, err = os.OpenFile(logFilepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666) //open file, if not exist, create at logFilepath
		if err != nil {
			panic(fmt.Sprintf("log file '%v' open failed: %v", logFilepath, err.Error()))
		}
	}

	stdLogger := &Logger{
		logger:   log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile),
		enabled:  enableStd,
		logLevel: Debug,
	}

	fileLogger := &Logger{
		logger:   log.New(outfile, "\r\n", log.Ldate|log.Ltime|log.Lshortfile),
		enabled:  enableFile,
		logLevel: Debug,
	}

	logger := &CombinedLogger{
		stdLogger:  stdLogger,
		fileLogger: fileLogger,
	}

	logger.stdLogger.logger.SetPrefix("[Info]")

	return logger

}

func NewLogger(filepath string, enableStd, enableFile bool) *CombinedLogger {
	return newLogger(filepath, enableStd, enableFile)
}

func NewDefaultLogger(filepath string, enableStd, enableFile bool) *CombinedLogger {
	TxLogger = newLogger(filepath, enableStd, enableFile)
	return TxLogger
}

func init() {
	clear := " "
	level2String[Detail] = "[DETAIL]" + clear
	level2String[Debug] = "[DEBUG]" + clear
	level2String[Info] = "[INFO]" + clear
	level2String[Warn] = "[WARN]" + clear
	level2String[Error] = "[ERROR]" + clear
	level2String[Fatal] = "[FATAL]" + clear
	//MyLogger = newLogger("./tmp/logs/stdout.log", true, true)
}

// SetLogLevel
func (l *Logger) SetLogLevel(lv LogLevel) {
	l.logLevel = lv
}

// GetLevelString
func (l *Logger) GetLevelString(lv LogLevel) string {
	if str, ok := level2String[lv]; ok {
		return str
	} else {

		return "[" + strconv.Itoa(int(lv)) + "]"
	}
}

func (l *Logger) SetEnabled(b bool) {
	l.enabled = b
}

// SetLogLevel
func (l *CombinedLogger) SetLogLevel(lv LogLevel) {
	l.stdLogger.SetLogLevel(lv)
	l.fileLogger.SetLogLevel(lv)
}

func (l *CombinedLogger) SetEnablestd(b bool) {
	l.stdLogger.SetEnabled(b)
}

func (l *CombinedLogger) SetEnablefile(b bool) {
	l.fileLogger.SetEnabled(b)
}

func (l *Logger) Log(level LogLevel, v ...interface{}) {
	if level < l.logLevel {
		return
	}

	if l.enabled {
		l.logger.SetPrefix(l.GetLevelString(level))
		//l.stdLogger.Println(v...)
		_ = l.logger.Output(3, fmt.Sprintln(v...))
	}
}

func (l *Logger) LogDepth(level LogLevel, calldepth int, v ...interface{}) {
	if level < l.logLevel {
		return
	}

	if l.enabled && l.logger != nil {
		l.logger.SetPrefix(l.GetLevelString(level))
		//l.stdLogger.Println(v...)
		_ = l.logger.Output(calldepth, fmt.Sprintln(v...))
	}
}

func (l *CombinedLogger) Log(level LogLevel, v ...interface{}) {
	if l.stdLogger != nil {
		l.stdLogger.Log(level, v...)
	}
	if l.fileLogger != nil {
		l.fileLogger.Log(level, v...)
	}
}

func (l *CombinedLogger) LogDepth(level LogLevel, calldepth int, v ...interface{}) {
	if l.stdLogger != nil {
		l.stdLogger.LogDepth(level, calldepth, v...)
	}
	if l.fileLogger != nil {
		l.fileLogger.LogDepth(level, calldepth, v...)
	}
}

func (l *CombinedLogger) ErrorLog(v ...interface{}) {
	//l.Log(Error, v...)
	l.LogDepth(Error, 4, v...)
}

// Log calls default logger and output info log
func Log(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Info, 4, v...)
}

func Logf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Info, 4, fmt.Sprintf(template, v...))
}

// ErrorLog call default logger and output error log
func ErrorLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Error, 4, v...)
}

func WarnLog(v ...interface{}) {
	TxLogger.LogDepth(Warn, 4, v...)
}

// ErrorLogf call default logger and output error log
func ErrorLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Error, 4, fmt.Errorf(template, v...))
}

// DebugLog calls default logger and output debug log
func DebugLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Debug, 4, v...)
}

// DebugLog calls default logger and output debug log
func DebugLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Debug, 4, fmt.Sprintf(template, v...))
}

// DebugLog calls default logger and output debug log
func DebugLogfWithCalldepth(calldepth int, template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Debug, calldepth, fmt.Sprintf(template, v...))
}

// DetailLog calls default logger and output detail log
func DetailLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Detail, 4, v...)
}

// DetailLog calls default logger and output detail log
func DetailLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TxLogger.LogDepth(Detail, 4, fmt.Sprintf(template, v...))
}

func FatalLogfAndExit(exitCode int, template string, v ...interface{}) {
	TxLogger.LogDepth(Fatal, 4, fmt.Sprintf(template, v...))
	os.Exit(exitCode)
}

// CheckError  TODO This is a bad way to call error log, as you cannot know where this method is called in your error log
// This give log line like this : [ERROR]2021/04/13 22:39:11 log.go:150: Fatal error: address 127.0.0.1: missing port in address
// it always refer to this file and this line
// If time allows, a better logging tool like "zerolog" can be used to replace these methods.
func CheckError(err error) bool {
	if err != nil {
		// a fatal error , should be fatal and exit. If that is not the expected behavior, change this log
		TxLogger.ErrorLog("Fatal error:", err.Error())
		return true
	}
	return false
}
