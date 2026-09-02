package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Point explicitly to the /api/generate endpoint
const ollamaURL = "http://host.docker.internal:11434/api/generate"

// Switched to standard gemma model to fix the manifest error on your 1070 Ti
const Gmodel = "gemma3-70gpu"

// Define the request structure matching Ollama's /api/generate format
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// Define the response structure matching Ollama's /api/generate format
type GenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"` // /api/generate puts text here instead of "message"
	Done     bool   `json:"done"`
}

// AskGemma passes the prompt directly to Ollama and returns the text reply
func AskGemma(userMessage string) (string, error) {
	reply, err := callOllama(userMessage)
	if err != nil {
		return "", err
	}
	return reply, nil
}

func callOllama(prompt string) (string, error) {
	// 1. Build the payload using the global Gmodel constant
	reqBody := GenerateRequest{
		Model:  Gmodel,
		Prompt: prompt,
		Stream: false,
	}

	// 2. Marshal payload into JSON bytes
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// 3. Send the POST request to the Generate API
	resp, err := http.Post(
		ollamaURL,
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return "", err
	}
	defer func() error {
		err := resp.Body.Close()
		if err != nil {
			return err
		}
		return nil

	}()
	// 4. Validate HTTP response code
	if resp.StatusCode != http.StatusOK {
		// Try to read error body if available to give you better hints
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	// 5. Decode response JSON straight into the struct
	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 6. Return the raw string answer text
	return result.Response, nil
}
