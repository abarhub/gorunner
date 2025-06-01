package main

import (
	"bufio"
	"fmt"
	"github.com/goccy/go-yaml"
	"gorunner/noSleep"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	////TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	//// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>
	//s := "gopher"
	//fmt.Printf("Hello and welcome, %s!\n", s)
	//
	//for i := 1; i <= 5; i++ {
	//	//TIP <p>To start your debugging session, right-click your code in the editor and select the Debug option.</p> <p>We have set one <icon src="AllIcons.Debugger.Db_set_breakpoint"/> breakpoint
	//	// for you, but you can always add more by pressing <shortcut actionId="ToggleLineBreakpoint"/>.</p>
	//	fmt.Println("i =", 100/i)
	//}

	//	yml := `
	//%YAML 1.2
	//---
	//a: 1
	//b: c
	//`
	//var v struct {
	//	A int
	//	B string
	//}
	//if err := yaml.Unmarshal([]byte(yml), &v); err != nil {
	//	fmt.Printf("error: %v\n", err)
	//} else {
	//	fmt.Printf("a: %v\n", v.A)
	//}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Démarrage du programme\n", timestamp)

	type Task struct {
		Run string
	}

	var param struct {
		Global struct {
			NoSleep bool `yaml:"no_sleep"`
		}
		Tasks []Task
	}

	data, err := os.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(data, &param); err != nil {
		fmt.Printf("error: %v\n", err)
	} else {

		// 2. Préparer le fichier de log
		logFileName := "program_output.log"
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Erreur lors de l'ouverture du fichier de log %s : %v", logFileName, err)
		}
		defer logFile.Close()

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] NoSleep: %v\n", timestamp, param.Global.NoSleep)

		if param.Global.NoSleep {
			go noSleep.PasSleep()
		}

		if len(param.Tasks) > 0 {
			for _, task := range param.Tasks {
				//fmt.Println(task.Run)

				run(task.Run, logFile)
			}
		}

	}

}

func run(run string, file *os.File) {

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
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Exécution de la commande : %s %s\n", timestamp, command, strings.Join(args, " "))
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
			// Formater la date et l'heure
			timestamp := time.Now().Format("2006-01-02 15:04:05") // Format personnalisable

			// Préfixer la ligne avec la date et l'heure
			formattedLine := fmt.Sprintf("[%s] %s%s", timestamp, prefix, line)

			// Afficher sur la console
			fmt.Println(formattedLine)
			// Écrire dans le fichier de log (avec un saut de ligne)
			_, err := file.WriteString(formattedLine + "\n")
			if err != nil {
				log.Printf("ATTENTION: Erreur lors de l'écriture dans le fichier de log : %v", err)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Erreur lors de la lecture du flux %s : %v", prefix, err)
		}
	}

	// Lancer les goroutines
	go processOutput(stdoutPipe, "")      // Pour la sortie standard
	go processOutput(stderrPipe, "ERR: ") // Pour la sortie d'erreur (on peut ajouter un préfixe distinctif)

	// 7. Attendre que la commande se termine
	err = cmd.Wait() // cmd.Wait() attend la fin de l'exécution et collecte le code de sortie
	if err != nil {
		// Afficher l'erreur dans la console et dans le log (si la commande a échoué)
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		errorMsg := fmt.Sprintf("[%s] Erreur lors de l'exécution de la commande : %v\n", timestamp, err)
		fmt.Print(errorMsg)
		_, logErr := file.WriteString(errorMsg)
		if logErr != nil {
			log.Printf("ATTENTION: Erreur lors de l'écriture de l'erreur dans le fichier de log : %v", logErr)
		}
		log.Fatalf("Erreur fatale lors de l'exécution : %v", err)
	}

	timestamp = time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Commande terminée\n", timestamp)
}
