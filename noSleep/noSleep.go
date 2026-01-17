package noSleep

import (
	"fmt"
	"gorunner/logutils"
	"syscall"
	"time"
)

// Execution States
const (
	// ES_SYSTEM_REQUIRED force le système à rester actif (empêche la mise en veille)
	ES_SYSTEM_REQUIRED = 0x00000001
	// ES_CONTINUOUS informe le système que l'état doit rester en vigueur jusqu'au prochain appel
	ES_CONTINUOUS = 0x80000000
)

var pulseTime = 10 * time.Second

var arret = make(chan bool)

func PasSleep() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setThreadExecStateProc := kernel32.NewProc("SetThreadExecutionState")

	pulse := time.NewTicker(pulseTime)

	logutils.Printf("Starting keep alive poll... (silence)")
	for {
		select {
		case <-arret:
			logutils.Printf("arret du no sleep ...")
			changementEtatSleep(setThreadExecStateProc, false)
			logutils.Printf("Fin du no sleep")
			return
		case <-pulse.C:
			changementEtatSleep(setThreadExecStateProc, true)
		}
	}
}

func FinNoSleep() {
	arret <- true
}

func changementEtatSleep(setThreadExecStateProc *syscall.LazyProc, veille bool) {
	var flags uint32
	if veille {
		flags = ES_CONTINUOUS | ES_SYSTEM_REQUIRED
	} else {
		flags = ES_CONTINUOUS
	}
	_, _, err := setThreadExecStateProc.Call(uintptr(flags))
	if err != nil {
		s := fmt.Sprintf("%v", err)
		if s != "L’opération a réussi." && s != "The operation completed successfully." {
			logutils.Printf("Erreur pour changer l'état de veille: %v", err)
		}
	}
}
