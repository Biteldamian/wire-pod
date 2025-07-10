package ai_proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

// Proxy handles forwarding to AI and parsing responses
type Proxy struct {
	Client *http.Client
}

// NewProxy creates a new proxy instance
func NewProxy() *Proxy {
	return &Proxy{
		Client: &http.Client{},
	}
}

// SendPrompt forwards speech text to AI endpoint
func (p *Proxy) SendPrompt(text string, events map[string]string) (string, error) {
	if Config.Endpoint == "dummy" {
		// Dummy for testing: echo back with TTS command
		return fmt.Sprintf("Echo: %s", text), nil
	}

	// Build request body (adapt for Ollama/OpenAI/custom)
	body := map[string]interface{}{
		"model":  Config.Model,
		"prompt": text,
		"events" = events  // Add to request body
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", Config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if Config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+Config.APIKey)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("AI endpoint error: " + resp.Status)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var aiResp map[string]interface{}
	json.Unmarshal(respBody, &aiResp)

	// Parse response (assume "response" field; adapt per API)
	if response, ok := aiResp["response"].(string); ok {
		return response, nil
	}
	return "", errors.New("invalid AI response format")
}

// ParseResponseToCommand parses AI text to basic commands (e.g., SayText)
func (p *Proxy) ParseResponseToCommand(aiText string) string {
	// For Sprint 1: simple TTS command
	// For Sprint 2: Assume aiText is JSON command; return for dispatch
	return aiText // Directly use as SayText param
				 // Dispatch happens in kgsim.go now
}
