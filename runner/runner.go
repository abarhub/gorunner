package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gorunner/config"
	"gorunner/logutils"
	"gorunner/metrics"
	"gorunner/noSleep"
	"gorunner/stat"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const EtatEnCour = "en_cours"
const EtatFin = "fin"
const MDCTache = "tache"
const RESUME = "resume"

func Run(param config.Parametres, logger *logutils.Logger) {

	debut := time.Now()

	if param.Global.NoSleep {
		go noSleep.PasSleep(logger)
		defer noSleep.FinNoSleep()
	}

	if param.Global.AttendDebutSecondes > 0 {
		logger.Printf("Attente %d secondes ...", param.Global.AttendDebutSecondes)
		time.Sleep(time.Duration(param.Global.AttendDebutSecondes) * time.Second)
		logger.Printf("Attente terminée")
	}

	err := ecrireEtat(param, EtatEnCour, logger)
	if err != nil {
		return
	}

	if len(param.Tasks) > 0 {
		stat2 := stat.CreateStat()
		for _, task := range param.Tasks {
			tache := stat.CreateExecutionTache()
			logger.AddMdc(MDCTache, task.NameId)
			if task.Enable {
				tache.Execute = true
				debut2 := time.Now()
				err := run(task, logger)
				diff := time.Now().Sub(debut2)
				tache.Duree = diff
				if err != nil {
					tache.Erreur = true
					logger.Printf("Erreur pour la tache %s : %v", task.Name, err)
				}
				metrics.AjoutMetrics(task.NameId, !tache.Erreur, diff.Seconds(), logger)
			} else {
				tache.Execute = false
				logger.Printf("Tache %s ignore", task.Name)
			}
			logger.DeleteMdc(MDCTache)
			stat2.Put(task.Name, tache)
		}

		logger.AddMdc(MDCTache, RESUME)
		logger.Printf("Résumé :")
		var dureeTotale time.Duration = 0
		var nbErreurs = 0
		var nbTaches = 0
		for _, taskName := range stat2.Keys() {
			task := stat2.Get(taskName)
			if task.Execute {
				logger.Printf("Tache %s : duree=%v, erreur=%v", taskName, formateDuration(task.Duree), task.Erreur)
				dureeTotale += task.Duree
				nbTaches++
				if task.Erreur {
					nbErreurs++
				}
			} else {
				logger.Printf("Tache %s : non executé", taskName)
			}

		}
		messageTelegrame := fmt.Sprintf("Nb taches=%d, duree totale=%v, nb erreur=%v.", nbTaches, formateDurationTelegrame(dureeTotale), nbErreurs)
		envoieTelegrame(param, messageTelegrame, logger)
	}

	if param.Global.AttendFinSecondes > 0 {
		logger.Printf("Attente %d secondes ...", param.Global.AttendFinSecondes)
		time.Sleep(time.Duration(param.Global.AttendFinSecondes) * time.Second)
		logger.Printf("Attente terminée")
	}

	diff := time.Now().Sub(debut)
	logger.Printf("Duree totale de toutes les taches : %v", formateDuration(diff))
	metrics.AjoutMetricsDuree(RESUME, diff.Seconds(), logger)
	logger.DeleteMdc(MDCTache)

	err = ecrireEtat(param, EtatFin, nil)
	if err != nil {
		return
	}
}

func formateDurationTelegrame(duree time.Duration) string {
	s := formateDuration(duree)
	if duree > time.Minute {
		return "<u><strong>" + s + "</strong></u>"
	}

	return s
}

func envoieTelegrame(param config.Parametres, message string, logger *logutils.Logger) {
	if param.Global.TelegramActive && len(param.Global.TelegramUrl) > 0 {

		url := fmt.Sprintf("%sbot%s/sendMessage", param.Global.TelegramUrl, param.Global.TelegramToken)
		values := map[string]string{"chat_id": param.Global.TelegrameBotToken, "text": message, "parse_mode": "HTML"}

		jsonValue, _ := json.Marshal(values)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

		if err != nil {
			logger.Printf("Erreur pour envoyer le message à telegrame %s", err.Error())
		} else {
			logger.Printf("Message envoyé à telegram avec succes (%d)", resp.StatusCode)
		}
	} else {
		logger.Printf("Pas d'envoi vers telegrame")
	}

}

func formateDuration(duration time.Duration) string {
	if duration.Milliseconds() > 1 {
		return duration.Truncate(10 * time.Millisecond).String()
	} else {
		return duration.String()
	}
}

func ecrireEtat(param config.Parametres, etat string, logger *logutils.Logger) error {
	if param.Global.EtatFile != "" {
		err := os.WriteFile(param.Global.EtatFile, []byte(etat), 0644)
		if err != nil {
			logger.Errorf("Erreur pour écrire dans le fichier %s : %v", param.Global.EtatFile, err)
			return err
		}
	}
	return nil
}

func run(task config.Task, logger *logutils.Logger) error {

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

	logger.Printf("Début de la tache %s", task.Name)

	debut := time.Now()

	// 3. Préparer la commande à exécuter
	cmd := exec.Command(command, args2...)

	// 4. Obtenir les Pipes pour Stdout et Stderr
	// Nous allons lire la sortie de la commande via ces pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logger.Errorf("Erreur lors de la création du pipe pour Stdout : %v", err)
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.Errorf("Erreur lors de la création du pipe pour Stderr : %v", err)
		return err
	}

	// 5. Démarrer la commande en arrière-plan
	// Nous utilisons Start() au lieu de Run() car nous voulons lire les pipes
	// pendant que la commande est en cours d'exécution.
	logger.Printf("Exécution de la commande : %s %s", command, strings.Join(args, " "))
	err = cmd.Start()
	if err != nil {
		logger.Errorf("Erreur lors du démarrage de la commande : %v", err)
		return err
	}

	// 6. Goroutines pour lire la sortie standard et d'erreur ligne par ligne
	// On utilise des goroutines pour lire les deux pipes en parallèle,
	// afin d'éviter un blocage si l'un des flux est rempli.

	// Lancer les goroutines
	go processOutput(stdoutPipe, "", task.Encoding, logger)      // Pour la sortie standard
	go processOutput(stderrPipe, "ERR: ", task.Encoding, logger) // Pour la sortie d'erreur (on peut ajouter un préfixe distinctif)

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
			logger.Errorf("Erreur lors de l'exécution de la commande : %v", err)
		}
	}

	logger.Printf("Commande terminée, status code : %d, durée : %v", statusCode, diff)

	logger.Printf("Fin de la tache %s", task.Name)
	return err
}

func contains(list []int, code int) bool {
	return slices.Contains(list, code)
}

func processOutput(reader io.Reader, prefix string, encoding string, logger *logutils.Logger) {
	reader2 := bufio.NewReader(reader)

	// Un buffer pour reconstituer la ligne morceau par morceau en mémoire
	var lineBuffer bytes.Buffer

	for {
		// Lire jusqu'au prochain '\n'
		fragment, err := reader2.ReadBytes('\n')

		// On accumule le morceau lu, qu'il soit partiel ou complet
		lineBuffer.Write(fragment)

		if err != nil {
			// 1. Fin du fichier atteinte
			if errors.Is(err, io.EOF) {
				// S'il reste des données dans le buffer, c'est la dernière ligne du fichier (sans '\n')
				if lineBuffer.Len() > 0 {
					traiterLigne(lineBuffer.Bytes(), prefix, encoding, logger)
				}
				break
			}

			// 2. Le tampon interne est plein, mais la ligne continue
			if errors.Is(err, bufio.ErrBufferFull) {
				// Le fragment a déjà été écrit dans lineBuffer, on continue la boucle
				// pour lire la suite de la même ligne.
				continue
			}

			// 3. Autre erreur de lecture fatale
			_, err2 := fmt.Fprintln(os.Stderr, "Erreur de lecture inattendue :", err)
			if err2 != nil {
				return
			}
			break
		}

		// Si err == nil, cela signifie qu'on a atteint le '\n'.
		// La ligne est entièrement reconstituée dans lineBuffer !
		traiterLigne(lineBuffer.Bytes(), prefix, encoding, logger)

		// On réinitialise le buffer pour la ligne suivante
		lineBuffer.Reset()
	}

}

func traiterLigne(ligne []byte, prefix string, encoding string, logger *logutils.Logger) {
	line := string(ligne)
	if encoding != "" {
		line = convertie(line, encoding, logger)
	}
	line = strings.TrimRight(line, "\r\n")

	// Afficher sur la console
	logger.Printf("%s%s", prefix, line)
}

func convertie(line string, encoding string, logger *logutils.Logger) string {
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
	} else if encoding == "UTF-8" {
		return line
	} else {
		logger.Errorf("Type d'encodage non géré: %s", encoding)
		return line
	}

}

// decodeWindows1252ToUTF8 convertit une chaîne encodée en Windows-1252 en UTF-8.
func decodeWindows1252ToUTF8(s string) (string, error) {
	reader := strings.NewReader(s)
	decoder := charmap.Windows1252.NewDecoder()
	transformedReader := transform.NewReader(reader, decoder)
	readAll, err := io.ReadAll(transformedReader)
	if err != nil {
		return "", err
	}
	return string(readAll), nil
}

// decodeISO88591ToUTF8 convertit une chaîne encodée en ISO-8859-1 en UTF-8.
func decodeISO88591ToUTF8(s string) (string, error) {
	reader := strings.NewReader(s)
	decoder := charmap.ISO8859_1.NewDecoder() // Utilisation de charmap.ISO8859_1
	transformedReader := transform.NewReader(reader, decoder)
	readAll, err := io.ReadAll(transformedReader)
	if err != nil {
		return "", err
	}
	return string(readAll), nil
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
