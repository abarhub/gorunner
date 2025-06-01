package noSleep

import (
	"fmt"
	"log"
	"syscall"
	"time"
)

// Execution States
const (
	EsSystemRequired = 0x00000001
	EsContinuous     = 0x80000000
)

var pulseTime = 10 * time.Second

func PasSleep() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setThreadExecStateProc := kernel32.NewProc("SetThreadExecutionState")

	pulse := time.NewTicker(pulseTime)

	log.Println("Starting keep alive poll... (silence)")
	for {
		select {
		case <-pulse.C:
			_, _, err := setThreadExecStateProc.Call(uintptr(EsSystemRequired))
			if err != nil {
				s := fmt.Sprintf("%v", err)
				if s != "L’opération a réussi." {
					log.Printf("Erreur pour changer l'état de veille: %v\n", err)
				}
			}
		}
	}
}
