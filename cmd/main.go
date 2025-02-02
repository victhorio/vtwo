package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openai/openai-go"
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
		fmt.Printf("[$%.2f] > ", app.GetCost())
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Error reading input: ", err)
		}
		input = strings.TrimSpace(input)

		if input == ":q" {
			break
		}

		history = append(history, openai.UserMessage(input))
		response := app.SendMessageStreaming(history)
		history = append(history, openai.AssistantMessage(response))
	}

	fmt.Printf("\nTotal cost for this session was $%.2f.\n", app.GetCost())
}
