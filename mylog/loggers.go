package mylog

import (
	"path/filepath"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"fmt"
	"os"
	"io"
	"github.com/annchain/OG/common/filename"
	"path"
)

var TxLogger *logrus.Logger

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func initLogger(logdir string, outputFile string, stdout bool, level logrus.Level) *logrus.Logger{
	var writer io.Writer

	if logdir != "" {
		folderPath, err := filepath.Abs(logdir)
		panicIfError(err, fmt.Sprintf("Error on parsing log path: %s", logdir))

		abspath, err := filepath.Abs(path.Join(logdir, outputFile))
		panicIfError(err, fmt.Sprintf("Error on parsing log file path: %s", logdir))

		err = os.MkdirAll(folderPath, os.ModePerm)
		panicIfError(err, fmt.Sprintf("Error on creating log dir: %s", folderPath))

		logrus.WithField("path", abspath).Info("Additional logger")
		logFile, err := os.OpenFile(abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		panicIfError(err, fmt.Sprintf("Error on creating log file: %s", abspath))

		writer = io.MultiWriter(logFile, logrus.StandardLogger().Writer())
	} else {
		// stdout only
		fmt.Println("Will be logged to stdout")
		writer = io.MultiWriter(os.Stdout, logrus.StandardLogger().Writer())
	}


	Formatter := new(logrus.TextFormatter)
	Formatter.ForceColors = stdout
	//Formatter.DisableColors = true
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logger := &logrus.Logger{
		Level:level,
		Formatter:Formatter,
		Out:writer,
	}

	lineNum := viper.GetBool("log_line_number")
	if lineNum {
		filenameHook := filename.NewHook()
		filenameHook.Field = "line"
		logrus.AddHook(filenameHook)
	}
	return logger
}

func InitLoggers(){
	logdir := viper.GetString("datadir")
	TxLogger = initLogger(logdir, "tx.log", true, logrus.DebugLevel)

	logrus.Debug("Additional logger initialized.")
}
