package vtwo

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openai/openai-go"
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
		client:  newAIClient(userConfig.API.BaseUrl, userConfig.API.ApiKey),
		rootCtx: context.Background(),
		timeout: time.Duration(userConfig.API.Timeout) * time.Second,

		model:           userConfig.API.Model,
		baseCost:        baseCost,
		outputCostRatio: userConfig.API.OutputCostRatio,

		rootDir: userConfig.Notes.BasePath,
	}
}
