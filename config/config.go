package config

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"sync"
)

type Task struct {
	Run string
}

type Parametres struct {
	Global struct {
		NoSleep bool   `yaml:"no_sleep"`
		LogFile string `yaml:"log_file"`
	}
	Tasks []Task
}

var param Parametres
var once sync.Once

func initConfig() {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}

	var param2 Parametres
	if err := yaml.Unmarshal(data, &param2); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	} else {
		param = param2
	}
}

func GetConfig() Parametres {
	once.Do(initConfig)

	return param
}
