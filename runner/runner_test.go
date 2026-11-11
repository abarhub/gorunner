package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func Test_processOutput(t *testing.T) {
	type args struct {
		reader   io.Reader
		prefix   string
		encoding string
		resultat string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{"test1", args{reader: strings.NewReader("Bonjour, Alice"), prefix: "", encoding: "UTF-8", resultat: "Bonjour, Alice\n"}},
		//{"test2", args{reader: strings.NewReader("Test 1,2,3 XXX"), prefix: "", encoding: "UTF-8", resultat: "Test 1,2,3 XXX\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			oldArgs := os.Args

			defer func() { os.Args = oldArgs }()

			tmpFile, err := os.CreateTemp("", "test-*.yml")
			if err != nil {
				t.Fatalf("Impossible de créer le fichier temporaire : %v", err)
			}

			tmpFile2, err := os.CreateTemp("", "test-log-*.log.txt")
			if err != nil {
				t.Fatalf("Impossible de créer le fichier temporaire de log : %v", err)
			}

			file := tmpFile2.Name()
			file = strings.ReplaceAll(file, "\\", "/")
			_, err = tmpFile.WriteString("global:\n   log_file: " + file)
			if err != nil {
				t.Fatalf("Erreur lors de l'ecritre dans le fichier' : %v", err)
			}

			// S'assurer de le supprimer à la fin du test
			defer os.Remove(tmpFile.Name())
			_ = tmpFile.Close()

			// S'assurer de le supprimer à la fin du test
			defer os.Remove(tmpFile2.Name())
			_ = tmpFile2.Close()

			// 2. Utiliser le chemin du fichier temporaire dans votre test
			cheminTest := tmpFile.Name()

			os.Args = []string{"mon-programme", cheminTest}

			oldStdout := os.Stdout

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Erreur lors de la création du pipe : %v", err)
			}

			processOutput(tt.args.reader, tt.args.prefix, tt.args.encoding, nil)

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			if err != nil {
				t.Fatalf("Erreur lors de la lecture du pipe : %v", err)
			}

			// Fermer le lecteur du pipe
			_ = r.Close()

			output := buf.String()

			contenuBytes, err := os.ReadFile(tmpFile2.Name())
			if err != nil {
				fmt.Fprintln(os.Stderr, "Erreur lors de la lecture du fichier :", err)
				return
			}

			// 2. Convertir les octets en string si vous en avez besoin
			contenuString := string(contenuBytes)
			output = contenuString

			// 6. Vérifier que la sortie contient le texte attendu
			texteAttendu := tt.args.resultat
			if !strings.HasSuffix(output, texteAttendu) {
				t.Errorf("Sortie attendue contenant %q, mais obtenu : %q", texteAttendu, output)
			}
		})
	}
}
