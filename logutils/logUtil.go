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

type Logger struct {
	out     io.Writer
	mu      sync.Mutex
	mdc     map[string]string
	logFile *os.File
}

func New(param config.Parametres) *Logger {

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
	outputWriter := io.MultiWriter(os.Stdout, logFile)
	mdc := make(map[string]string)

	return &Logger{out: outputWriter, mdc: mdc, logFile: logFile}
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	var msg3 = ""
	if len(l.mdc) > 0 {
		var msg2 strings.Builder
		for key, value := range l.mdc {
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

	_, err := fmt.Fprint(l.out, final)
	if err != nil {
		l.Errorf("Erreur de formatage: %v", err)
		return
	}
}

func (l *Logger) Fermeture() {
	if l.logFile != nil {

		err := l.logFile.Close()
		if err != nil {
			return
		}
	}
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

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Printf("ERROR: "+format, args...)
}

func (l *Logger) AddMdc(key, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mdc[key] = value
}

func (l *Logger) DeleteMdc(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.mdc, key)
}
