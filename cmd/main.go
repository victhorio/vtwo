package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/victhorio/vtwo/vtwo"
)

func main() {
	app := vtwo.NewApp()
	repl(app)
}

func repl(app *vtwo.VTwo) {
	reader := bufio.NewReader(os.Stdin)
	history := vtwo.NewChatHistory()

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Error reading input: ", err)
		}
		input = strings.TrimSpace(input)

		if input == ":q" {
			break
		}

		var resp string
		resp, history, err = app.SendMessage(input, history)
		if err != nil {
			log.Fatal("Error getting model response: ", err)
		}

		fmt.Printf("\n[$%.2f] V2: %s\n\n", app.GetCost(), resp)
	}

	fmt.Printf("Total cost for this session was %.2f.\n", app.GetCost())
}
