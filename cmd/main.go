package main

import (
	"fmt"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/runner"
	"time"
)

func main() {

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Démarrage du programme\n", timestamp)

	param := config.GetConfig()

	logutils.Printf("initialisation")

	defer logutils.CloseLog()

	logutils.Printf("NoSleep: %v", param.Global.NoSleep)

	runner.Run(param)

	logutils.Printf("fin de l'execution des taches")
}
