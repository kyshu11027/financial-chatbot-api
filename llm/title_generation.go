package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
)

const openaiURL = "https://api.openai.com/v1/chat/completions"

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Message Message `json:"message"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

func GenerateChatTitle(userMessage string) (string, error) {
	reqBody := OpenAIRequest{
		Model:       "gpt-3.5-turbo",
		MaxTokens:   20,
		Temperature: 0.3,
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant that generates short, descriptive titles for financial advice chat conversations. Keep it under 5 words using only alphanumeric characters."},
			{Role: "user", Content: fmt.Sprintf("Create a short title for this chat: %q", userMessage)},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", err
	}

	if len(openaiResp.Choices) > 0 {
		return cleanString(openaiResp.Choices[0].Message.Content), nil
	}
	return "New Chat", nil
}

func cleanString(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9 ':,;-]+`)
	cleaned := re.ReplaceAllString(input, "")
	return cleaned
}
