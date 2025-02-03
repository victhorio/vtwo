package vtwo

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// VTwo holds contextual information needed between function calls
type VTwo struct {
	client  *openai.Client
	rootCtx context.Context
	timeout time.Duration

	model                     string
	baseCost, outputCostRatio float64

	usageMutex                     sync.Mutex
	promptTokens, completionTokens int64

	rootDir string
}

// NewApp creates a new VTwo instance
func NewApp() *VTwo {
	userConfig := loadConfig()

	// don't like that this is here, but also don't want to keep this in the config
	// at least the output to input token ratio is constant per provider, so works well enough
	// don't know yet though
	// TODO: think about this?
	var costMap = map[string]float64{
		openai.ChatModelGPT4oMini: 0.15 / 1_000_000,
		openai.ChatModelGPT4o:     2.5 / 1_000_000,
		openai.ChatModelO3Mini:    1.1 / 1_000_000,
	}

	baseCost, ok := costMap[userConfig.API.Model]
	if !ok {
		log.Printf("WARNING: Unknown cost for model `%s`, assuming $2.50/1M prompt tokens.", userConfig.API.Model)
		baseCost = 2.5 / 1_000_000
	}

	return &VTwo{
		client:  newClient(userConfig.API.BaseUrl, userConfig.API.ApiKey),
		rootCtx: context.Background(),
		timeout: time.Duration(userConfig.API.Timeout) * time.Second,

		model:           userConfig.API.Model,
		baseCost:        baseCost,
		outputCostRatio: userConfig.API.OutputCostRatio,

		rootDir: userConfig.Notes.BasePath,
	}
}

func newClient(baseUrl, key string) *openai.Client {
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
