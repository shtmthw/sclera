package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func Dispatch(ctx context.Context, call *AIResponse) (string, error) {
	switch call.Tool {
	case "web_search":
		return dispatchWebSearch(ctx, call)

	default:
		return "", fmt.Errorf(
			"unknown tool call: %q",
			call.Tool,
		)
	}
}

func dispatchWebSearch(ctx context.Context, call *AIResponse) (string, error) {
	rawQuery, ok := call.Arguments["query"]

	if !ok {
		return "", fmt.Errorf(
			"web_search: missing query argument",
		)
	}

	query, ok := rawQuery.(string)
	if !ok {
		return "", fmt.Errorf(
			"web_search: query must be a string, got %T",
			rawQuery,
		)
	}

	query = strings.TrimSpace(query)

	if query == "" {
		return "", fmt.Errorf(
			"web_search: query cannot be empty",
		)
	}

	if len(query) > 1000 {
		return "", fmt.Errorf(
			"web_search: query is too long",
		)
	}

	return SearchWeb(ctx, query)
}

// Optional helper for logging/debugging tool requests.
func ToolCallJSON(call *AIResponse) string {
	data, err := json.Marshal(call)
	if err != nil {
		return ""
	}

	return string(data)
}
