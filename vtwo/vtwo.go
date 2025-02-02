package vtwo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var costMap = map[string]float64{
	openai.ChatModelGPT4oMini: 0.15 / 1_000_000,
	openai.ChatModelGPT4o:     2.5 / 1_000_000,
	openai.ChatModelO3Mini:    1.1 / 1_000_000,
}

const model = openai.ChatModelGPT4oMini
const completionToPromptCostRatio = 4.0

// VTwo holds contextual information needed between function calls
type VTwo struct {
	client  *openai.Client
	rootCtx context.Context

	usageMutex                     sync.Mutex
	promptTokens, completionTokens int64
}

// NewApp creates a new VTwo instance
func NewApp(apiKey string) *VTwo {
	const baseUrl = "https://api.openai.com/v1/"
	return &VTwo{
		client:  newClient(baseUrl, apiKey),
		rootCtx: context.Background(),
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
			Model:    openai.F(openai.ChatModelGPT4oMini),
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

	baseCost := costMap[model]
	promptCost := float64(v.promptTokens) * baseCost
	completionCost := float64(v.completionTokens) * baseCost * completionToPromptCostRatio
	return promptCost + completionCost
}
