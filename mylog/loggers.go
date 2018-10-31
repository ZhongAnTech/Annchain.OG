package mylog

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
	"path/filepath"
)

var TxLogger *logrus.Logger

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func InitLogger(logger *logrus.Logger , logdir string, outputFile string) *logrus.Logger {
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
       //write  a message to just one  files
		writer = io.MultiWriter(logFile)
	} else {
		// stdout only
		fmt.Println("Will be logged to stdout")
		writer = io.MultiWriter(os.Stdout)
	}
	 return  &logrus.Logger{
		Level:     logger.Level,
		Formatter: logger.Formatter,
		Out:       writer,
		Hooks:logger.Hooks,
	}
}

func InitLoggers( logger *logrus.Logger) {
	logdir := viper.GetString("datadir")
	TxLogger = InitLogger(logger,logdir, "tx.log",)
	logrus.Debug("Additional logger initialized.")
}

