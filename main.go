package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	//TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>
	s := "gopher"
	fmt.Printf("Hello and welcome, %s!\n", s)

	for i := 1; i <= 5; i++ {
		//TIP <p>To start your debugging session, right-click your code in the editor and select the Debug option.</p> <p>We have set one <icon src="AllIcons.Debugger.Db_set_breakpoint"/> breakpoint
		// for you, but you can always add more by pressing <shortcut actionId="ToggleLineBreakpoint"/>.</p>
		fmt.Println("i =", 100/i)
	}

	yml := `
%YAML 1.2
---
a: 1
b: c
`
	var v struct {
		A int
		B string
	}
	if err := yaml.Unmarshal([]byte(yml), &v); err != nil {
		fmt.Printf("error: %v\n", err)
	} else {
		fmt.Printf("a: %v\n", v.A)
	}

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
		fmt.Printf("NoSleep: %v\n", param.Global.NoSleep)
	}
}
