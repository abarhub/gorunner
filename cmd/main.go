package main

import (
	"bufio"
	"fmt"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/noSleep"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"
)

func main() {

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Démarrage du programme\n", timestamp)

	param := config.GetConfig()

	logutils.Printf("initialisation\n")

	defer logutils.CloseLog()

	logutils.Printf("NoSleep: %v\n", param.Global.NoSleep)

	if param.Global.NoSleep {
		go noSleep.PasSleep()
	}

	if len(param.Tasks) > 0 {
		for _, task := range param.Tasks {
			run(task.Run)
		}
	}

}

func run(run string) {

	stringSlice := strings.Split(run, " ")

	command := stringSlice[0]
	args := stringSlice[1:]

	// 3. Préparer la commande à exécuter
	cmd := exec.Command(command, args...)

	// 4. Obtenir les Pipes pour Stdout et Stderr
	// Nous allons lire la sortie de la commande via ces pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Erreur lors de la création du pipe pour Stdout : %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Erreur lors de la création du pipe pour Stderr : %v", err)
	}

	// 5. Démarrer la commande en arrière-plan
	// Nous utilisons Start() au lieu de Run() car nous voulons lire les pipes
	// pendant que la commande est en cours d'exécution.
	logutils.Printf("Exécution de la commande : %s %s\n", command, strings.Join(args, " "))
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Erreur lors du démarrage de la commande : %v", err)
	}

	// 6. Goroutines pour lire la sortie standard et d'erreur ligne par ligne
	// On utilise des goroutines pour lire les deux pipes en parallèle,
	// afin d'éviter un blocage si l'un des flux est rempli.

	// Fonction utilitaire pour lire un flux et loguer
	processOutput := func(reader io.Reader, prefix string) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// Afficher sur la console
			logutils.Printf("%s%s\n", prefix, line)

		}
		if err := scanner.Err(); err != nil {
			logutils.Printf("Erreur lors de la lecture du flux %s : %v\n", prefix, err)
		}
	}

	// Lancer les goroutines
	go processOutput(stdoutPipe, "")      // Pour la sortie standard
	go processOutput(stderrPipe, "ERR: ") // Pour la sortie d'erreur (on peut ajouter un préfixe distinctif)

	// 7. Attendre que la commande se termine
	err = cmd.Wait() // cmd.Wait() attend la fin de l'exécution et collecte le code de sortie
	if err != nil {
		// Afficher l'erreur dans la console et dans le log (si la commande a échoué)
		logutils.Printf("Erreur lors de l'exécution de la commande : %v\n", err)
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		log.Fatalf("[%s] Erreur fatale lors de l'exécution : %v", timestamp, err)
	}

	logutils.Printf("Commande terminée\n")
}
