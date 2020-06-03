package cmd

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/arefactor/common/goroutine"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"time"
)

func DumpStack() {
	if err := recover(); err != nil {
		logrus.WithField("obj", err).Error("Fatal error occurred. Program will exit")
		var buf bytes.Buffer
		stack := debug.Stack()
		buf.WriteString(fmt.Sprintf("Panic: %v\n", err))
		buf.Write(stack)
		dumpName := "dump_" + time.Now().Format("20060102-150405")
		nerr := ioutil.WriteFile(dumpName, buf.Bytes(), 0644)
		if nerr != nil {
			fmt.Println("write dump file error", nerr)
			fmt.Println(buf.String())
		}
		logrus.WithField("stack ", buf.String()).Error("panic")
	}
}

// initLogger uses viper to get the log path and level. It should be called by all other commands
func initLogger() {
	logdir := viper.GetString("logdir")
	stdout := viper.GetBool("log_stdout")

	var writer io.Writer

	if logdir != "" {
		folderPath, err := filepath.Abs(logdir)
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on parsing log path: %s", logdir))

		abspath, err := filepath.Abs(path.Join(logdir, "run"))
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on parsing log file path: %s", logdir))

		err = os.MkdirAll(folderPath, os.ModePerm)
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on creating log dir: %s", folderPath))

		if stdout {
			logFile, err := os.OpenFile(abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			utilfuncs.PanicIfError(err, fmt.Sprintf("Error on creating log file: %s", abspath))
			abspath += ".log"
			fmt.Println("Will be logged to stdout and ", abspath)
			writer = io.MultiWriter(os.Stdout, logFile)
		} else {
			fmt.Println("Will be logged to ", abspath+".log")
			writer = mylog.RotateLog(abspath)
		}
	} else {
		// stdout only
		fmt.Println("Will be logged to stdout")
		writer = os.Stdout
	}

	logrus.SetOutput(writer)

	// Only log the warning severity or above.
	switch viper.GetString("log_level") {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		fmt.Println("Unknown level", viper.GetString("log_level"), "Set to INFO")
		logrus.SetLevel(logrus.InfoLevel)
	}

	Formatter := new(logrus.TextFormatter)
	Formatter.ForceColors = logdir == ""
	//Formatter.DisableColors = true
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logrus.SetFormatter(Formatter)

	// redirect standard log to logrus
	//log.SetOutput(logrus.StandardLogger().Writer())
	//log.Println("Standard logger. Am I here?")
	lineNum := viper.GetBool("log_line_number")
	if lineNum {
		//filenameHook := filename.NewHook()
		//filenameHook.Field = "line"
		//logrus.AddHook(filenameHook)
		logrus.SetReportCaller(true)
	}
	byLevel := viper.GetBool("multifile_by_level")
	if byLevel && logdir != "" {
		panicLog, _ := filepath.Abs(path.Join(logdir, "panic"))
		fatalLog, _ := filepath.Abs(path.Join(logdir, "fatal"))
		warnLog, _ := filepath.Abs(path.Join(logdir, "warn"))
		errorLog, _ := filepath.Abs(path.Join(logdir, "error"))
		infoLog, _ := filepath.Abs(path.Join(logdir, "info"))
		debugLog, _ := filepath.Abs(path.Join(logdir, "debug"))
		traceLog, _ := filepath.Abs(path.Join(logdir, "trace"))
		writerMap := lfshook.WriterMap{
			logrus.PanicLevel: mylog.RotateLog(panicLog),
			logrus.FatalLevel: mylog.RotateLog(fatalLog),
			logrus.WarnLevel:  mylog.RotateLog(warnLog),
			logrus.ErrorLevel: mylog.RotateLog(errorLog),
			logrus.InfoLevel:  mylog.RotateLog(infoLog),
			logrus.DebugLevel: mylog.RotateLog(debugLog),
			logrus.TraceLevel: mylog.RotateLog(traceLog),
		}
		logrus.AddHook(lfshook.NewHook(
			writerMap,
			Formatter,
		))
	}
	logger := logrus.StandardLogger()
	logrus.Debug("Logger initialized.")
	byModule := viper.GetBool("multifile_by_module")
	if !byModule {
		logdir = ""
	}

	downloader.InitLoggers(logger, logdir)
	fetcher.InitLoggers(logger, logdir)
	p2p.InitLoggers(logger, logdir)
	og.InitLoggers(logger, logdir)
	syncer.InitLoggers(logger, logdir)
	annsensus.InitLoggers(logger, logdir)
}

func startPerformanceMonitor() {
	function := func() {
		logrus.WithField("port", viper.GetString("profiling.port")).Info("Performance monitor started")
		log.Println(http.ListenAndServe("0.0.0.0:"+viper.GetString("profiling.port"), nil))
	}
	goroutine.New(function)
}
