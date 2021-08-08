package logger

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	logFileName string = "budget.log"
)

var (
	logFileHandle *os.File
	log           *logrus.Logger = logrus.New()
)

func init() {
	log.SetLevel(logrus.InfoLevel) // TODO Configure in config file
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

func SetLoggingToDirectory(logDir string) {
	loggingFileAbsPath := path.Join(logDir, logFileName)
	log.Infof("Logging to file %s", loggingFileAbsPath)
	/* #nosec */
	logFileHandle, err := os.OpenFile(
		loggingFileAbsPath,
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0600)
	if err != nil {
		log.Fatalf("Error opening file %v\n", err)
	}
	log.SetOutput(logFileHandle)
}

func CloseLogFile() {
	if ^logFileHandle.Fd() == 0 {
		return
	}

	if err := logFileHandle.Close(); err != nil {
		fmt.Println("Couldn't close logging file handle")
		log.Error("Couldn't close logging file handle")
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
