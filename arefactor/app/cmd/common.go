package cmd

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/arefactor/common/files"
	"github.com/annchain/OG/arefactor/common/mylog"
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

var (
	LogDir     = "log"
	DataDir    = "data"
	ConfigDir  = "config"
	PrivateDir = "private"
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
		fmt.Println(buf.String())
	}
}

// initLogger uses viper to get the log path and level. It should be called by all other commands
func initLogger() {
	doStdout := viper.GetBool("log-stdout")
	doFile := viper.GetBool("log-file")
	logdir := files.FixPrefixPath(viper.GetString("rootdir"), LogDir)

	var writers []io.Writer

	if doFile {
		folderPath, err := filepath.Abs(logdir)
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on parsing log path: %s", logdir))

		abspath, err := filepath.Abs(path.Join(logdir, "run"))
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on parsing log file path: %s", logdir))

		err = os.MkdirAll(folderPath, os.ModePerm)
		utilfuncs.PanicIfError(err, fmt.Sprintf("Error on creating log dir: %s", folderPath))
		writers = append(writers, mylog.RotateLog(abspath))
		fmt.Println("Will be logged to " + abspath + ".log")
	}
	if doStdout {
		fmt.Println("Will be logged to stdout")
		writers = append(writers, os.Stdout)
	}

	if doStdout && doFile {
		writer := io.MultiWriter(writers...)
		logrus.SetOutput(writer)
	} else if doStdout {
		logrus.SetOutput(os.Stdout)
	}

	// Only log the warning severity or above.
	switch viper.GetString("log-level") {
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
		fmt.Println("Unknown level: ", viper.GetString("log-level"), "Set to INFO")
		logrus.SetLevel(logrus.InfoLevel)
	}

	Formatter := new(logrus.TextFormatter)
	Formatter.ForceColors = doStdout
	//Formatter.DisableColors = false
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logrus.StandardLogger().SetFormatter(Formatter)

	// redirect standard log to logrus
	//log.SetOutput(logrus.StandardLogger().Writer())
	//log.Println("Standard logger. Am I here?")
	lineNum := viper.GetBool("log-line-number")
	if lineNum {
		//filenameHook := filename.NewHook()
		//filenameHook.Field = "line"
		//logrus.AddHook(filenameHook)
		logrus.SetReportCaller(true)
	}
	byLevel := viper.GetBool("multifile_by_level")
	if byLevel && doFile {
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
	//logger := logrus.StandardLogger()
	logrus.Debug("Logger initialized.")
	byModule := viper.GetBool("multifile_by_module")
	if !byModule {
		logdir = ""
	}
}

func startPerformanceMonitor() {
	go func() {
		port := viper.GetString("profiling.port")
		logrus.WithField("port", port).Info("Performance monitor started")

		log.Println(http.ListenAndServe("0.0.0.0:"+port, nil))
	}()
}

func ensureFolder() {
	root := viper.GetString("rootdir")
	err := files.MkDirIfNotExists(root)
	utilfuncs.PanicIfError(err, "creating root folder")

	folders := []string{LogDir, DataDir, ConfigDir, PrivateDir}
	for _, folder := range folders {
		err = files.MkDirIfNotExists(files.FixPrefixPath(root, folder))
		utilfuncs.PanicIfError(err, "creating folder: "+folder)
	}
}
