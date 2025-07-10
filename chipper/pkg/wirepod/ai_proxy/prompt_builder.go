package ai_proxy

import (
	"encoding/json"
	"fmt"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

func BuildPrompt(speechText string) (string, error) {
	prompt := map[string]string{"query": speechText}
	jsonPrompt, err := json.Marshal(prompt)
	if err != nil {
		logger.Println("Prompt error: " + err.Error())
		return "", err
	}
	return fmt.Sprintf("Handle: %s", string(jsonPrompt)), nil
}