package ai_proxy

import (
	"encoding/json"
	"fmt"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

// BuildPrompt creates a basic JSON prompt from speech text
func BuildPrompt(speechText string) (string, error) {
	prompt := map[string]string{
		"query": speechText,
		// Extend with sensors/camera in later sprints
	}
	jsonPrompt, err := json.Marshal(prompt)
	if err != nil {
		logger.Println("Error building prompt: " + err.Error())
		return "", err
	}
	return fmt.Sprintf("Handle this query: %s", string(jsonPrompt)), nil
}