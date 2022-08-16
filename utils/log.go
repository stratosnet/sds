package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
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

var RpcLoggerMap = NewAutoCleanMap(60 * time.Minute)

type Logger struct {
	logger   *log.Logger
	enabled  bool
	logLevel LogLevel
}

type CombinedLogger struct {
	stdLogger  *Logger
	fileLogger *Logger
}

var MyLogger *CombinedLogger
var TrafficLogger *CombinedLogger
var RpcLogger *Logger

func newLogger(logFilepath string, enableStd, enableFile bool) *CombinedLogger {
	if err := os.MkdirAll(filepath.Dir(logFilepath), os.ModePerm); err != nil {
		panic(fmt.Sprintf("log file '%v' initialize failed: %v", logFilepath, err.Error()))
	}
	//init file output
	outfile, err := os.OpenFile(logFilepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666) //open file, if not exist, create at logFilepath
	if err != nil {
		panic(fmt.Sprintf("log file '%v' open failed: %v", logFilepath, err.Error()))
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
	MyLogger = newLogger(filepath, enableStd, enableFile)
	return MyLogger
}

func NewTrafficLogger(filePath string, enableStd, enableFile bool) *CombinedLogger {
	TrafficLogger = newLogger(filePath, enableStd, enableFile)
	return TrafficLogger
}

//Log calls default logger and output info log
func DumpTraffic(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	TrafficLogger.LogDepth(Info, 3, v...)
}

func GetLastLinesFromTrafficLog(path string, n uint64) []string {

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	line := ""
	lines := []string{}
	var cursor int64 = 0
	stat, _ := file.Stat()
	filesize := stat.Size()
	var i uint64
	for i = 0; i < n; i++ {
		for {
			cursor -= 1
			file.Seek(cursor, io.SeekEnd)

			char := make([]byte, 1)
			file.Read(char)

			if cursor != -1 && (char[0] == 10 || char[0] == 13) {
				break
			}

			line = fmt.Sprintf("%s%s", string(char), line)

			if cursor == -filesize {
				break
			}
		}
		lines = append(lines, line)
		if cursor == -filesize {
			break
		}
	}

	return lines
}

func init() {
	clear := "\033[0m"
	level2String[Detail] = "\033[0;32m[Detail]" + clear
	level2String[Debug] = "\033[0;36m[DEBUG]" + clear
	level2String[Info] = "\033[0;34m[INFO]" + clear
	level2String[Warn] = "\033[0;33m[WARN]" + clear
	level2String[Error] = "\033[0;35m[ERROR]" + clear
	level2String[Fatal] = "\033[0;31m[FATAL]" + clear
	//MyLogger = newLogger("./tmp/logs/stdout.log", true, true)
}

//SetLogLevel
func (l *Logger) SetLogLevel(lv LogLevel) {
	l.logLevel = lv
}

//GetLevelString
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

//SetLogLevel
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
		l.logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) LogDepth(level LogLevel, calldepth int, v ...interface{}) {
	if level < l.logLevel {
		return
	}

	if l.enabled && l.logger != nil {
		l.logger.SetPrefix(l.GetLevelString(level))
		//l.stdLogger.Println(v...)
		l.logger.Output(calldepth, fmt.Sprintln(v...))
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
	l.LogDepth(Error, 3, v...)
}

func SetRpcLogger(rpc io.Writer) {
	logLevel := Debug
	if MyLogger != nil && MyLogger.stdLogger != nil {
		logLevel = MyLogger.stdLogger.logLevel
	}

	RpcLogger = &Logger{
		logger:   log.New(rpc, "\r\n", log.Ldate|log.Ltime|log.Lshortfile),
		enabled:  true,
		logLevel: logLevel,
	}
}

func ClearRpcLogger() {
	RpcLogger.enabled = false
	RpcLogger.logger = nil
	RpcLoggerMap = NewAutoCleanMap(60 * time.Minute)
}

//Log calls default logger and output info log
func Log(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Info, 3, v...)
}

func Logf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Info, 3, fmt.Sprintf(template, v...))
}

//ErrorLog call default logger and output error log
func ErrorLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Error, 3, v...)
}

func WarnLog(v ...interface{}) {
	MyLogger.LogDepth(Warn, 3, v...)
}

//ErrorLogf call default logger and output error log
func ErrorLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Error, 3, fmt.Errorf(template, v...))
}

//DebugLog calls default logger and output debug log
func DebugLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Debug, 3, v...)
}

//DebugLog calls default logger and output debug log
func DebugLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Debug, 3, fmt.Sprintf(template, v...))
}

//DetailLog calls default logger and output detail log
func DetailLog(v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Detail, 3, v...)
}

//DetailLog calls default logger and output detail log
func DetailLogf(template string, v ...interface{}) {
	//GetLogger().Log(Info, v...)
	MyLogger.LogDepth(Detail, 3, fmt.Sprintf(template, v...))
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
