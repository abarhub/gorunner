package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/noSleep"
	"gorunner/stat"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const EtatEnCour = "en_cours"
const EtatFin = "fin"

func Run(param config.Parametres) {

	debut := time.Now()

	if param.Global.NoSleep {
		go noSleep.PasSleep()
		defer noSleep.FinNoSleep()
	}

	if param.Global.AttendDebutSecondes > 0 {
		logutils.Printf("Attente %d secondes ...", param.Global.AttendDebutSecondes)
		time.Sleep(time.Duration(param.Global.AttendDebutSecondes) * time.Second)
		logutils.Printf("Attente terminée")
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
		messageTelegrame := "Résumé :"
		for _, taskName := range stat2.Keys() {
			task := stat2.Get(taskName)
			if task.Execute {
				logutils.Printf("Tache %s : duree=%v, erreur=%v", taskName, task.Duree, task.Erreur)
				messageTelegrame += fmt.Sprintf("\nTache %s : duree=%v, erreur=%v.", taskName, task.Duree, task.Erreur)
			} else {
				logutils.Printf("Tache %s : non executé", taskName)
				messageTelegrame += fmt.Sprintf("\nTache %s : non executé.", taskName)
			}

		}
		envoieTelegrame(param, messageTelegrame)

	}

	diff := time.Now().Sub(debut)
	logutils.Printf("Duree totale de toutes les taches : %v", diff)

	err = ecrireEtat(param, EtatFin)
	if err != nil {
		return
	}
}

func envoieTelegrame(param config.Parametres, message string) {
	if len(param.Global.TelegramUrl) > 0 {

		url := fmt.Sprintf("%sbot%s/sendMessage", param.Global.TelegramUrl, param.Global.TelegramToken)
		values := map[string]string{"chat_id": param.Global.TelegrameBotToken, "text": message, "parse_mode": "HTML"}

		jsonValue, _ := json.Marshal(values)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

		if err != nil {
			logutils.Printf("Erreur pour envoyer le message à telegrame %s", err.Error())
		} else {
			logutils.Printf("Message envoyé à telegram avec succes (%d)", resp.StatusCode)
		}
	} else {
		logutils.Printf("Pas d'envoi vers telegrame")
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

	var command string
	var args []string

	if task.Run != "" {
		run := task.Run
		stringSlice := strings.Split(run, " ")

		command = stringSlice[0]
		args = stringSlice[1:]
	} else {
		command = task.Commands[0]
		args = task.Commands[1:]
	}

	var args2 []string

	argsTemplate := Args{Now: time.Now().Format("2006-01-02_15-04-05")}

	for _, arg := range args {
		if strings.Contains(arg, "{{") {
			arg, err := replace(arg, argsTemplate)
			if err != nil {
				return err
			}
			args2 = append(args2, arg)
		} else {
			args2 = append(args2, arg)
		}

	}

	logutils.Printf("Début de la tache %s", task.Name)

	debut := time.Now()

	// 3. Préparer la commande à exécuter
	cmd := exec.Command(command, args2...)

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
		if len(task.ExitCodeOk) > 0 && contains(task.ExitCodeOk, statusCode) {
			err = nil
		} else {
			logutils.Errorf("Erreur lors de l'exécution de la commande : %v", err)
		}
	}

	logutils.Printf("Commande terminée, status code : %d, durée : %v", statusCode, diff)

	logutils.Printf("Fin de la tache %s", task.Name)
	return err
}

func contains(list []int, code int) bool {
	for _, element := range list {
		if element == code {
			return true
		}
	}
	return false
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
