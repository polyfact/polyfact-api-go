package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"

	type_parser "github.com/polyfact/api/type_parser"
)

func generateTypedPrompt(type_format string, task string) string {
	return "Your goal is to write a JSON object that will accomplish a specific task.\nThe string inside the JSON must be plain text, and not contain any markdown or HTML unless explicitely mentionned in the task.\nThe JSON object should follow this type:\n```\n" + type_format + "\n``` The task you must accomplish:\n" + task + "\n\nPlease only provide the JSON in a single json markdown code block with the keys described above. Do not include any other text.\nPlease make sure the JSON is a single line and does not contain any newlines outside of the strings. The type must be strictly respected. Do not skip any of the fields. If a field is not relevant, use an empty string, a 0 or an empty array."
}

func removeMarkdownCodeBlock(s string) string {
	return strings.TrimPrefix(strings.Trim(strings.TrimSpace(s), "`"), "json")
}

type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

type Result struct {
	Result     any        `json:"result"`
	TokenUsage TokenUsage `json:"token_usage"`
}

func Generate(type_format any, task string, c *func(string, int, int)) (Result, error) {
	tokenUsage := TokenUsage{Input: 0, Output: 0}
	type_string, err := type_parser.TypeToString(type_format, 0)
	if err != nil {
		return Result{Result: "{\"error\":\"parse_type_failed\"}", TokenUsage: tokenUsage}, err
	}

	for i := 0; i < 5; i++ {
		log.Printf("Trying generation %d/5\n", i+1)
		llm, err := openai.NewChat()
		if err != nil {
			return Result{Result: "{\"error\":\"llm_init_failed\"}", TokenUsage: tokenUsage}, err
		}

		input_prompt := generateTypedPrompt(type_string, task)
		ctx := context.Background()
		completion, err := llm.Call(ctx, []schema.ChatMessage{
			schema.HumanChatMessage{Text: input_prompt},
		})
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		if c != nil {
			(*c)(os.Getenv("OPENAI_MODEL"), llm.GetNumTokens(input_prompt), llm.GetNumTokens(completion))
		}

		tokenUsage.Input += llm.GetNumTokens(input_prompt)
		tokenUsage.Output += llm.GetNumTokens(completion)

		result_json := removeMarkdownCodeBlock(completion)

		if !json.Valid([]byte(result_json)) {
			fmt.Printf("%v\n", result_json)
			continue
		}

		var result any
		json.Unmarshal([]byte(result_json), &result)

		type_check := type_parser.CheckAgainstType(type_format, result)

		if !type_check {
			fmt.Printf("%v\n", result)
			continue
		}

		return Result{Result: result, TokenUsage: tokenUsage}, err
	}

	return Result{Result: "{\"error\":\"generation_failed\"}", TokenUsage: tokenUsage}, errors.New("Generation failed after 5 retries")
}
