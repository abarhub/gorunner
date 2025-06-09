package runner

import (
	"bufio"
	"errors"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/noSleep"
	"gorunner/stat"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const EtatEnCour = "en_cours"
const EtatFin = "fin"

func Run(param config.Parametres) {
	if param.Global.NoSleep {
		go noSleep.PasSleep()
	}

	err := ecrireEtat(param, EtatEnCour)
	if err != nil {
		return
	}

	if len(param.Tasks) > 0 {
		stat2 := stat.CreateStat()
		for _, task := range param.Tasks {
			tache := stat.CreateExecutionTache()
			if task.Enable {
				tache.Execute = true
				debut := time.Now()
				err := run(task)
				diff := time.Now().Sub(debut)
				tache.Duree = diff
				if err != nil {
					tache.Erreur = true
					logutils.Printf("Erreur pour la tache %s : %v", task.Name, err)
				}
			} else {
				tache.Execute = false
				logutils.Printf("Tache %s ignore", task.Name)
			}
			stat2.Put(task.Name, tache)
		}
		logutils.Printf("Résumé :")
		for _, taskName := range stat2.Keys() {
			task := stat2.Get(taskName)
			if task.Execute {
				logutils.Printf("Tache %s : duree=%v, erreur=%v", taskName, task.Duree, task.Erreur)
			} else {
				logutils.Printf("Tache %s : non executé", taskName)
			}

		}

	}

	err = ecrireEtat(param, EtatFin)
	if err != nil {
		return
	}
}

func ecrireEtat(param config.Parametres, etat string) error {
	if param.Global.EtatFile != "" {
		err := os.WriteFile(param.Global.EtatFile, []byte(etat), 0644)
		if err != nil {
			logutils.Errorf("Erreur pour écrire dans le fichier %s : %v", param.Global.EtatFile, err)
			return err
		}
	}
	return nil
}

func run(task config.Task) error {

	run := task.Run
	stringSlice := strings.Split(run, " ")

	command := stringSlice[0]
	args := stringSlice[1:]

	logutils.Printf("Début de la tache %s", task.Name)

	debut := time.Now()

	// 3. Préparer la commande à exécuter
	cmd := exec.Command(command, args...)

	// 4. Obtenir les Pipes pour Stdout et Stderr
	// Nous allons lire la sortie de la commande via ces pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logutils.Errorf("Erreur lors de la création du pipe pour Stdout : %v", err)
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logutils.Errorf("Erreur lors de la création du pipe pour Stderr : %v", err)
		return err
	}

	// 5. Démarrer la commande en arrière-plan
	// Nous utilisons Start() au lieu de Run() car nous voulons lire les pipes
	// pendant que la commande est en cours d'exécution.
	logutils.Printf("Exécution de la commande : %s %s", command, strings.Join(args, " "))
	err = cmd.Start()
	if err != nil {
		logutils.Errorf("Erreur lors du démarrage de la commande : %v", err)
		return err
	}

	// 6. Goroutines pour lire la sortie standard et d'erreur ligne par ligne
	// On utilise des goroutines pour lire les deux pipes en parallèle,
	// afin d'éviter un blocage si l'un des flux est rempli.

	// Lancer les goroutines
	go processOutput(stdoutPipe, "", task.Encoding)      // Pour la sortie standard
	go processOutput(stderrPipe, "ERR: ", task.Encoding) // Pour la sortie d'erreur (on peut ajouter un préfixe distinctif)

	// 7. Attendre que la commande se termine
	err = cmd.Wait() // cmd.Wait() attend la fin de l'exécution et collecte le code de sortie
	statusCode := 0
	diff := time.Now().Sub(debut)
	if err != nil {
		// Afficher l'erreur dans la console et dans le log (si la commande a échoué)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				statusCode = status.ExitStatus()
			}
		}
		logutils.Errorf("Erreur lors de l'exécution de la commande : %v", err)
	}

	logutils.Printf("Commande terminée, status code : %d, durée : %v", statusCode, diff)

	logutils.Printf("Fin de la tache %s", task.Name)
	return err
}

func processOutput(reader io.Reader, prefix string, encoding string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if encoding != "" {
			line = convertie(line, encoding)
		}

		// Afficher sur la console
		logutils.Printf("%s%s", prefix, line)

	}
	if err := scanner.Err(); err != nil {
		logutils.Printf("Erreur lors de la lecture du flux %s : %v", prefix, err)
	}
}

func convertie(line string, encoding string) string {
	if encoding == "Windows1252" {
		s, err := decodeWindows1252ToUTF8(line)
		if err != nil {
			return line
		} else {
			return s
		}
	} else if encoding == "ISO88591" {
		s, err := decodeISO88591ToUTF8(line)
		if err != nil {
			return line
		} else {
			return s
		}
	} else {
		logutils.Errorf("Type d'encodage non géré: %s", encoding)
		return line
	}

}

// decodeWindows1252ToUTF8 convertit une chaîne encodée en Windows-1252 en UTF-8.
func decodeWindows1252ToUTF8(s string) (string, error) {
	reader := strings.NewReader(s)
	decoder := charmap.Windows1252.NewDecoder()
	transformedReader := transform.NewReader(reader, decoder)
	bytes, err := io.ReadAll(transformedReader)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// decodeISO88591ToUTF8 convertit une chaîne encodée en ISO-8859-1 en UTF-8.
func decodeISO88591ToUTF8(s string) (string, error) {
	reader := strings.NewReader(s)
	decoder := charmap.ISO8859_1.NewDecoder() // Utilisation de charmap.ISO8859_1
	transformedReader := transform.NewReader(reader, decoder)
	bytes, err := io.ReadAll(transformedReader)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
