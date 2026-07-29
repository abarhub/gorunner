package logutils

import (
	"bytes"
	"fmt"
	"gorunner/config"
	"io"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	outputWriter io.Writer
	once         sync.Once
	logFile      *os.File
	mdc          map[string]string
)

// initLogger initialise le writer partagé (stdout + fichier)
func initLogger() {

	param := config.GetConfig()

	if param.Global.LogFile == "" {
		fmt.Println("logFile is empty")
		os.Exit(1)
	}

	f := param.Global.LogFile
	if strings.Contains(f, "{{") {
		s, err := replace(f, Args{Now: time.Now().Format("2006-01-02")})
		if err != nil {
			fmt.Errorf("erreur: %v", err)
			os.Exit(1)
		}
		f = s
	} else if strings.HasSuffix(f, ".log") {
		f = strings.TrimSuffix(f, ".log")
		f = f + "." + time.Now().Format("2006-01-02") + ".log"
	}

	logFile, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur ouverture fichier log: %v\n", err)
		os.Exit(1)
	}
	outputWriter = io.MultiWriter(os.Stdout, logFile)
	mdc = make(map[string]string)
}

type Args struct {
	Now string
}

func replace(tmplt string, args Args) (string, error) {
	var buf bytes.Buffer
	t, err := template.New("tmp").Parse(tmplt)
	if err != nil {
		return "", err // as error?!
	}
	err = t.Execute(&buf, args)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
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
	var msg3 = ""
	if len(mdc) > 0 {
		var msg2 strings.Builder
		for key, value := range mdc {
			if msg2.Len() > 0 {
				msg2.WriteString(",")
			}
			msg2.WriteString(key + "=" + value)
		}
		msg3 = msg2.String()
		if len(msg3) > 0 {
			msg3 = "[" + msg3 + "]"
		}
	}
	final := fmt.Sprintf("[%s] %s %s\n", timestamp, msg3, msg)

	_, err := fmt.Fprint(outputWriter, final)
	if err != nil {
		Printf("Erreur de formatage: %v", err)
		return
	}
}

func Errorf(format string, args ...interface{}) {
	Printf("ERROR: "+format, args...)
}

func CloseLog() {
	once.Do(fermeture)
}

func AddMdc(key, value string) {
	mdc[key] = value
}

func DeleteMdc(key string) {
	delete(mdc, key)
}
