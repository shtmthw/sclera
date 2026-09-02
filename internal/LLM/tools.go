package llm

// "context"
// "fmt"

// Tool describes a single function Gemma is allowed to call. This mirrors
// the JSON shape Ollama's /api/chat endpoint expects in the "tools" field:
//
//	{
//	  "type": "function",
//	  "function": {
//	    "name": "...",
//	    "description": "...",
//	    "parameters": { "type": "object", "properties": {...}, "required": [...] }
//	  }
//	}
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

type ToolParameters struct {
	Type       string                  `json:"type"`
	Required   []string                `json:"required,omitempty"`
	Properties map[string]ToolProperty `json:"properties"`
}

type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// AvailableTools returns every tool Gemma is allowed to use. callOllama
// attaches this slice to every ChatRequest.
func AvailableTools() []Tool {
	return []Tool{
		webSearchTool(),
	}
}

func webSearchTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "web_search",
			Description: "Search the public internet for current information. " +
				"Use this only when the answer requires up-to-date facts " +
				"(current events, latest versions/releases, prices, or anything " +
				"that could have changed recently). Do not use it for general " +
				"knowledge you already know.",
			Parameters: ToolParameters{
				Type:     "object",
				Required: []string{"query"},
				Properties: map[string]ToolProperty{
					"query": {
						Type:        "string",
						Description: "The search query to run.",
					},
				},
			},
		},
	}
}

// Dispatch executes a tool call by name and returns the plain-text result
// that gets sent back to Gemma as a "tool" role message. Add a new case
// here whenever you add a tool to AvailableTools.
// func Dispatch(ctx context.Context, call ToolCall) (string, error) {
// 	switch call.Function.Name {
// 	case "web_search":
// 		query, ok := call.Function.Arguments["query"].(string)
// 		if !ok || query == "" {
// 			return "", fmt.Errorf("web_search: missing or invalid \"query\" argument")
// 		}
// 		return SearchWeb(ctx, query)
// 	default:
// 		return "", fmt.Errorf("unknown tool call: %q", call.Function.Name)
// 	}
// }
