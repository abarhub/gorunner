package main

import (
	"fmt"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/metrics"
	"gorunner/runner"
	"time"
)

func main() {

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Démarrage du programme\n", timestamp)

	param := config.NewDefaut(true)

	logger := logutils.New(param)

	go func() {
		metrics.Init(param, logger)
	}()

	logger.Printf("initialisation")

	defer logger.Fermeture()

	logger.Printf("NoSleep: %v", param.Global.NoSleep)

	runner.Run(param, logger)

	logger.Printf("fin de l'execution des taches")
}
