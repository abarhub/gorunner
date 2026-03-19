package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

type Task struct {
	Name       string
	Run        string
	Commands   []string
	Encoding   string
	Enable     bool
	ExitCodeOk []int `yaml:"exit_code_ok"`
}

type Parametres struct {
	Global struct {
		NoSleep             bool   `yaml:"no_sleep"`
		LogFile             string `yaml:"log_file"`
		EtatFile            string `yaml:"etat_file"`
		AttendDebutSecondes int    `yaml:"attente_debut_secondes"`
		TelegramToken       string `yaml:"telegram_token"`
		TelegrameBotToken   string `yaml:"telegrame_bot_token"`
		TelegramUrl         string `yaml:"telegram_url"`
	}
	Tasks []Task
}

var param Parametres
var once sync.Once

const ENV_FILE = ".env"

func trouveFichierEnv() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)

	envFile := filepath.Join(exPath, ENV_FILE)

	if _, err := os.Stat(envFile); !errors.Is(err, os.ErrNotExist) {
		return envFile
	}

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	envFile = filepath.Join(path, ENV_FILE)

	if _, err := os.Stat(envFile); !errors.Is(err, os.ErrNotExist) {
		return envFile
	}

	return ""
}

func chargeEnv() {

	envFile := trouveFichierEnv()

	if len(envFile) > 0 {
		fmt.Println("chargement du fichier:", envFile)
		err := godotenv.Load(envFile)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("pas de fichier .env")
	}

}

func initConfig() {

	chargeEnv()

	configFile := "config.yml"

	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	fmt.Println("configFile:", configFile)
	fmt.Println("len:", len(os.Args))

	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	expanded := os.ExpandEnv(string(data))

	var param2 Parametres
	if err := yaml.Unmarshal([]byte(expanded), &param2); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	} else {
		for i, t := range param2.Tasks {
			if t.Name == "" {
				t.Name = fmt.Sprintf("task-%d", i)
			}
		}
		param = param2
	}
}

func GetConfig() Parametres {
	once.Do(initConfig)

	return param
}
