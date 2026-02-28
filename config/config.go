package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/goccy/go-yaml"
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

func initConfig() {

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

	var param2 Parametres
	if err := yaml.Unmarshal(data, &param2); err != nil {
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
