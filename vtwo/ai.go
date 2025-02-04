package vtwo

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func newAIClient(baseUrl, key string) *openai.Client {
	return openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithAPIKey(key),
		option.WithMaxRetries(3),
	)
}

// NewChatHistory is a utility function to create an empty history slice
func NewChatHistory() []openai.ChatCompletionMessageParamUnion {
	return []openai.ChatCompletionMessageParamUnion{}
}

// SendMessage sends a message to the API and returns the response text, given a history with an
// already appended user message at the end.
func (v *VTwo) SendMessage(history []openai.ChatCompletionMessageParamUnion) (string, error) {
	ctx, cancel := context.WithTimeout(v.rootCtx, v.timeout)
	defer cancel()

	chatCompletion, err := v.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(v.model),
			Messages: openai.F(history),
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		return "", err
	}

	responseText := chatCompletion.Choices[0].Message.Content
	return responseText, nil
}

// SendMessageStreaming is used for user interaction for immediate feedback. Given a history
// with an already appended user message at the end, it sends a message to the API and receives
// the response in chunks, immediately printing each chunk to os.Stdout. It also returns the
// complete response as a string.
func (v *VTwo) SendMessageStreaming(history []openai.ChatCompletionMessageParamUnion) string {
	ctx, cancel := context.WithTimeout(v.rootCtx, v.timeout)
	defer cancel()

	params := openai.ChatCompletionNewParams{
		Messages:      openai.F(history),
		Model:         openai.F(v.model),
		StreamOptions: openai.F(openai.ChatCompletionStreamOptionsParam{IncludeUsage: openai.F(true)}),
	}
	stream := v.client.Chat.Completions.NewStreaming(ctx, params)

	printCtx := &printModelCtx{}

	acc := openai.ChatCompletionAccumulator{}
	fmt.Printf("\n")
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if _, ok := acc.JustFinishedContent(); ok {
			fmt.Printf("\n\n")
		} else if refusal, ok := acc.JustFinishedRefusal(); ok {
			fmt.Printf("\n[Model refused to answer]\n\n%s\n", refusal)
		} else if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.Content) > 0 {
			printModelContent(printCtx, chunk.Choices[0].Delta.Content)
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalf("Error on model stream response: %s\n", err.Error())
	}

	v.UpdateUsage(acc.Usage.PromptTokens, acc.Usage.CompletionTokens)
	return acc.Choices[0].Message.Content
}

type printModelCtx struct {
	boldPrefix bool
	bold       bool
}

func printModelContent(ctxt *printModelCtx, content string) {
	// for now this is pretty much a regular printing job, except that once we
	// hit a "**" we replace it with a "\x1b[1;32m" (bold and green) until we
	// get another "**" and then we replace it with "\x1b[22;39m" (reset bold,
	// default color)

	for _, ch := range content {
		if ch == '*' {
			if ctxt.boldPrefix {
				ctxt.boldPrefix = false
				ctxt.bold = !ctxt.bold
				if ctxt.bold {
					fmt.Printf("\x1b[1;32m")
				} else {
					fmt.Printf("\x1b[22;39m")
				}
			} else {
				ctxt.boldPrefix = true
			}
		} else {
			if ctxt.boldPrefix {
				// we were waiting for a "**" but got something else, wo we owe
				// an asterisk to stdout
				ctxt.boldPrefix = false
				fmt.Print("*")
			}
			fmt.Printf("%c", ch)
		}
	}
}

func (v *VTwo) UpdateUsage(promptTokens, completionTokens int64) {
	v.usageMutex.Lock()
	defer v.usageMutex.Unlock()

	v.promptTokens += promptTokens
	v.completionTokens += completionTokens
}

func (v *VTwo) GetCost() float64 {
	v.usageMutex.Lock()
	defer v.usageMutex.Unlock()

	promptCost := float64(v.promptTokens) * v.baseCost
	completionCost := float64(v.completionTokens) * v.baseCost * v.outputCostRatio
	return promptCost + completionCost
}
