package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	logFileHandle *os.File
	log           *logrus.Logger = logrus.New()
)

func init() {
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
		PadLevelText:  false,
		ForceQuote:    false,
	})
}

// funcCallTracer traces function calls so we can know where
// log lines are originated.
// I couldn't get logrus.TextFormatter's attribute
// CallerPrettyfier to work as expected.
func funcCallTracer() *logrus.Entry {
	pc, filename, lineNro, ok := runtime.Caller(2)
	var funcName string
	if !ok {
		filename = "?"
		lineNro = 0
	} else {
		funcName = runtime.FuncForPC(pc).Name()
	}

	filename = filepath.Base(filename)
	funcName = filepath.Base(funcName)
	return log.WithFields(logrus.Fields{
		"filename": filename,
		"line":     lineNro,
		"func":     funcName,
	})
}

func SetLoggingToFile(logFilePath string) error {
	cleanedPath, err := filepath.Abs(logFilePath)
	if err != nil {
		return err
	}
	loggingFileAbsPath := filepath.Clean(cleanedPath)
	log.Infof("Logging to file %s", loggingFileAbsPath)
	logFileHandle, err := os.OpenFile(
		loggingFileAbsPath,
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0o600)
	if err != nil {
		return err
	}
	log.SetOutput(logFileHandle)

	return nil
}

func CloseLogFile() {
	if ^logFileHandle.Fd() == 0 {
		return
	}

	if err := logFileHandle.Close(); err != nil {
		fmt.Println("Couldn't close logging file handle")
		Error("Couldn't close logging file handle")
	}
}

func Debug(args ...interface{}) {
	funcCallTracer().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	funcCallTracer().Debugf(format, args...)
}

func Error(args ...interface{}) {
	funcCallTracer().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	funcCallTracer().Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	funcCallTracer().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	funcCallTracer().Fatalf(format, args...)
}

func Info(args ...interface{}) {
	funcCallTracer().Info(args...)
}

func Infof(format string, args ...interface{}) {
	funcCallTracer().Infof(format, args...)
}

func Panic(args ...interface{}) {
	funcCallTracer().Panic(args...)
}

func Panicf(format string, args ...interface{}) {
	funcCallTracer().Panicf(format, args...)
}

func Warn(args ...interface{}) {
	funcCallTracer().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	funcCallTracer().Warnf(format, args...)
}
