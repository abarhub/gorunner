package logutils

import (
	"fmt"
	"gorunner/config"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	outputWriter io.Writer
	once         sync.Once
	logFile      *os.File
)

// initLogger initialise le writer partagé (stdout + fichier)
func initLogger() {

	param := config.GetConfig()

	if param.Global.LogFile == "" {
		fmt.Println("logFile is empty")
		os.Exit(1)
	}

	f := param.Global.LogFile
	if strings.HasSuffix(f, ".log") {
		f = strings.TrimSuffix(f, ".log")
		f = f + "." + time.Now().Format("2006-01-02") + ".log"
	}

	logFile, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur ouverture fichier log: %v\n", err)
		os.Exit(1)
	}
	outputWriter = io.MultiWriter(os.Stdout, logFile)
}

func fermeture() {
	if logFile != nil {

		err := logFile.Close()
		if err != nil {
			return
		}
	}
}

// Printf écrit un message horodaté dans la console et le fichier
func Printf(format string, args ...interface{}) {
	once.Do(initLogger) // S'assure que l'init est faite une seule fois

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	final := fmt.Sprintf("[%s] %s\n", timestamp, msg)

	fmt.Fprint(outputWriter, final)
}

func Errorf(format string, args ...interface{}) {
	Printf("ERROR: "+format, args...)
}

func CloseLog() {
	once.Do(fermeture)
}
