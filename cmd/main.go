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
	// repl(app)
	app.TrackFiles()
}

func repl(app *vtwo.VTwo) {
	reader := bufio.NewReader(os.Stdin)
	history := vtwo.NewChatHistory()

	for running := true; running; {
		fmt.Printf("\x1b[1;35m>\x1b[39m ")
		input, err := reader.ReadString('\n')
		fmt.Printf("\x1b[0m")
		if err != nil {
			log.Fatal("Error reading input: ", err)
		}
		input = strings.TrimSpace(input)

		if input[0] == ':' {
			switch input {
			case ":q", ":quit": // quit
				running = false
			case ":c", ":clear": // clear
				fmt.Printf("\nClearing chat history.\n\n")
				history = vtwo.NewChatHistory()
			case ":u", ":undo": // undo
				fmt.Printf("\nRemoving last interaction.\n\n")
				// since each interaction consists of a user message and an assistant message,
				// we remove the last two messages from the history
				history = history[:len(history)-2]
			default:
				fmt.Printf("Unknown command: %s\n", input)
			}
		} else {
			history = append(history, openai.UserMessage(input))
			response := app.SendMessageStreaming(history)
			history = append(history, openai.AssistantMessage(response))
		}
	}

	finalCost := app.GetCost()
	if finalCost >= 0.01 {
		fmt.Printf("\nTotal cost for this session was $%.2f.\n", finalCost)
	} else if finalCost > 0.0001 {
		fmt.Printf("\nTotal cost for this session was $%.4f.\n", finalCost)
	}
}
