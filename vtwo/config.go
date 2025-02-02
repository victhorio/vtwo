package vtwo

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type UserConfig struct {
	API   APIConfig   `json:"api_config"`
	Notes NotesConfig `json:"notes_config"`
}

type APIConfig struct {
	BaseUrl         string  `json:"base_url"`
	ApiKey          string  `json:"api_key"`
	Model           string  `json:"model"`
	OutputCostRatio float64 `json:"output_cost_ratio"`
}

type NotesConfig struct {
	BasePath string `json:"base_path"`
}

func loadConfig() UserConfig {
	// TODO: should the program really fatal here?

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not find user's home directory: %s\n", err.Error())
	}
	configPath := filepath.Join(homeDir, ".v2", "config.json")

	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Could not open config file: %s\n", err.Error())
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var config UserConfig
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Could not parse config file: %s\n", err.Error())
	}

	return config
}
