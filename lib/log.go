package lib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogDirectory = "logs"
	LogFileName  = "log"
)

/*
	This file implements a logging system with support for different log levels (Debug, Info, Warn, Error, Fatal) and colored output.
	The Logger can output logs to various destinations, including stdout and **auto-rotating of log files**, and provides formatted logging methods.
*/

func init() {
	color.NoColor = false
}

// LoggerI defines the interface for various logging levels and formatted output
type LoggerI interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	Print(msg string)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Printf(format string, args ...interface{})
}

const (
	DebugLevel int32 = -4
	InfoLevel  int32 = 0
	WarnLevel  int32 = 4
	ErrorLevel int32 = 8

	Reset = iota
	RED
	GREEN
	YELLOW
	BLUE
	GRAY
)

var (
	_ LoggerI = &Logger{}
)

// LoggerConfig holds configuration settings for the logger, including logging level and output writer
type LoggerConfig struct {
	Level int32 `json:"level"`
	Out   io.Writer
}

// Logger is the concrete implementation of LoggerI, managing log output based on configuration
type Logger struct {
	config LoggerConfig
}

// Debug() logs a message at the Debug level with blue color
func (l *Logger) Debug(msg string) {
	if l.config.Level <= DebugLevel {
		l.write(colorString(BLUE, "DEBUG: "+msg))
	}
}

// Info() logs a message at the Info level with green color
func (l *Logger) Info(msg string) {
	if l.config.Level <= InfoLevel {
		l.write(colorString(GREEN, "INFO: "+msg))
	}
}

// Warn() logs a message at the Warn level with yellow color
func (l *Logger) Warn(msg string) {
	if l.config.Level <= WarnLevel {
		l.write(colorString(YELLOW, "WARN: "+msg))
	}
}

// Error() logs a message at the Error level with red color
func (l *Logger) Error(msg string) {
	if l.config.Level <= ErrorLevel {
		l.write(colorString(RED, "ERROR: "+msg))
	}
}

// Print() logs a message without any specific log level or color
func (l *Logger) Print(msg string) { l.write(msg) }

// Fatal() logs an error message and terminates the program
func (l *Logger) Fatal(msg string) {
	l.Error(msg)
	os.Exit(1)
}

// Debugf() logs a formatted message at the Debug level with blue color
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.config.Level <= DebugLevel {
		l.write(colorStringWithFormat(BLUE, "DEBUG: "+format, args...))
	}
}

// Infof() logs a formatted message at the Info level with green color
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.config.Level <= InfoLevel {
		l.write(colorStringWithFormat(GREEN, "INFO: "+format, args...))
	}
}

// Warnf() logs a formatted message at the Warn level with yellow color
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.config.Level <= WarnLevel {
		l.write(colorStringWithFormat(YELLOW, "WARN: "+format, args...))
	}
}

// Errorf() logs a formatted message at the Error level with red color
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.config.Level <= ErrorLevel {
		l.write(colorStringWithFormat(RED, "ERROR: "+format, args...))
	}
}

// Fatalf() logs a formatted error message and terminates the program
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.write(colorStringWithFormat(RED, "FATAL: "+format, args...))
	os.Exit(1)
}

// Printf() logs a formatted message without any specific log level or color
func (l *Logger) Printf(format string, args ...interface{}) {
	l.write(fmt.Sprintf(format, args...))
}

// write() outputs the log message with a timestamp to the configured writer
func (l *Logger) write(msg string) {
	timeColored := colorString(GRAY, time.Now().Format(time.StampMilli))
	if _, err := l.config.Out.Write([]byte(fmt.Sprintf("%s %s\n", timeColored, msg))); err != nil {
		fmt.Println(newLogError(err))
	}
}

// NewLogger() creates a new Logger instance with the specified configuration and optional data directory path
func NewLogger(config LoggerConfig, dataDirPath ...string) LoggerI {
	if config.Out == nil {
		if dataDirPath == nil || dataDirPath[0] == "" {
			dataDirPath = make([]string, 1)
			dataDirPath[0] = DefaultDataDirPath()
		}
		logPath := filepath.Join(dataDirPath[0], LogDirectory, LogFileName)
		if _, err := os.Stat(logPath); errors.Is(err, os.ErrNotExist) {
			if err = os.MkdirAll(filepath.Join(dataDirPath[0], LogDirectory), os.ModePerm); err != nil {
				panic(err)
			}
		}
		logFile := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    30, // megabyte
			MaxBackups: 1500,
			MaxAge:     14, // days
			Compress:   true,
		}
		config.Out = io.MultiWriter(os.Stdout, logFile)
	}
	return &Logger{
		config: config,
	}
}

// NewDefaultLogger() creates a Logger with default settings, logging at the Debug level to stdout
func NewDefaultLogger() LoggerI {
	return NewLogger(LoggerConfig{
		Level: DebugLevel,
		Out:   os.Stdout,
	})
}

// NewNullLogger() creates a Logger that discards all log output
func NewNullLogger() LoggerI {
	return NewLogger(LoggerConfig{
		Level: DebugLevel,
		Out:   io.Discard,
	})
}

// colorStringWithFormat() returns a formatted string with color applied
func colorStringWithFormat(c int, format string, args ...interface{}) string {
	return colorString(c, fmt.Sprintf(format, args...))
}

// colorString() returns a string with color applied, preserving line breaks
func colorString(c int, msg string) (res string) {
	arr := strings.Split(msg, "\n")
	l := len(arr)
	for i, part := range arr {
		res += cString(c, part)
		if i != l-1 {
			res += "\n"
		}
	}
	return
}

// cString() returns a string with a specific color applied
func cString(c int, msg string) string {
	switch c {
	case BLUE:
		return color.BlueString(msg)
	case RED:
		return color.RedString(msg)
	case YELLOW:
		return color.YellowString(msg)
	case GREEN:
		return color.GreenString(msg)
	case GRAY:
		return color.HiBlackString(msg)
	default:
		return color.WhiteString(msg)
	}
}
