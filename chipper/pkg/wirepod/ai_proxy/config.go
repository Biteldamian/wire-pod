package ai_proxy

import (
	"os"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"gopkg.in/yaml.v3"
)

type AIConfig struct {
	Endpoint string `yaml:"endpoint"` // e.g., "http://localhost:11434/api/generate" for Ollama
	APIKey   string `yaml:"api_key"`  // Optional, for API-based models
	Model    string `yaml:"model"`    // e.g., "llama2" for Ollama, "gpt-3.5-turbo" for OpenAI
	Mode     string `yaml:"mode"`     // "full-control" or "speech-only"
}

var Config AIConfig

// LoadConfig loads AI settings from YAML/ENV, extending vars/config.go
func LoadConfig() error {
	// Default to speech-only mode for Sprint 1
	Config.Mode = "speech-only"

	// Load from YAML if exists (extend vars.APIConfig loading logic)
	data, err := os.ReadFile(vars.ApiConfigPath)
	if err == nil {
		var fullConfig struct {
			AI AIConfig `yaml:"ai"`
		}
		if err := yaml.Unmarshal(data, &fullConfig); err != nil {
			return err
		}
		Config = fullConfig.AI
		logger.Println("Loaded AI config from YAML")
	}

	// Override with ENV vars if set
	if envEndpoint := os.Getenv("AI_ENDPOINT"); envEndpoint != "" {
		Config.Endpoint = envEndpoint
	}
	if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		Config.APIKey = envKey
	}
	if envModel := os.Getenv("AI_MODEL"); envModel != "" {
		Config.Model = envModel
	}
	if envMode := os.Getenv("AI_MODE"); envMode != "" {
		Config.Mode = envMode
	}

	if Config.Endpoint == "" {
		logger.Println("AI_ENDPOINT not set; using dummy echo for testing")
		Config.Endpoint = "dummy"
	}

	return nil
}

// SaveConfig saves AI config back to YAML, integrating with vars.WriteConfigToDisk()
func SaveConfig() error {
	// For now, log save; integrate with full config write later
	logger.Println("Saving AI config")
	return nil
}