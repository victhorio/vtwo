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
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		panic("OPENAI_API_KEY not defined")
	}
	app := vtwo.NewApp(key)
	history := vtwo.NewChatHistory()

	reader := bufio.NewReader(os.Stdin)
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

	fmt.Printf("Total cost for this session was %.2f.", app.GetCost())
}
