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

	model                     string
	baseCost, outputCostRatio float64

	usageMutex                     sync.Mutex
	promptTokens, completionTokens int64
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

		model:           userConfig.API.Model,
		baseCost:        baseCost,
		outputCostRatio: userConfig.API.OutputCostRatio,
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

// SendMessage sends a message to the OpenAI API with a given history and returns
// the response text as well as the updated history.
func (v *VTwo) SendMessage(message string, history []openai.ChatCompletionMessageParamUnion) (string, []openai.ChatCompletionMessageParamUnion, error) {
	ctx, cancel := context.WithTimeout(v.rootCtx, 5*time.Minute)
	defer cancel()

	newHistory := append(history, openai.UserMessage(message))
	chatCompletion, err := v.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(v.model),
			Messages: openai.F(newHistory),
		},
		option.WithRequestTimeout(1*time.Minute),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		return "", history, err
	}

	responseText := chatCompletion.Choices[0].Message.Content
	newHistory = append(newHistory, openai.AssistantMessage(responseText))

	return responseText, newHistory, nil
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
