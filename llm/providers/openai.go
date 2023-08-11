package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	goOpenai "github.com/sashabaranov/go-openai"
	"github.com/tmc/langchaingo/llms"
)

const OPENAI_MODEL = "gpt-3.5-turbo-16k"

type OpenAIStreamProvider struct {
	client goOpenai.Client
}

func NewOpenAIStreamProvider() OpenAIStreamProvider {
	config := goOpenai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
	config.OrgID = os.Getenv("OPENAI_ORGANIZATION")
	return OpenAIStreamProvider{
		client: *goOpenai.NewClientWithConfig(config),
	}
}

func (m OpenAIStreamProvider) Generate(task string, c *func(string, int, int), opts *ProviderOptions) chan Result {
	chan_res := make(chan Result)

	go func() {
		defer close(chan_res)
		tokenUsage := TokenUsage{Input: 0, Output: 0}
		for i := 0; i < 5; i++ {
			log.Printf("Trying generation %d/5\n", i+1)

			ctx := context.Background()

			req := goOpenai.ChatCompletionRequest{
				Model: goOpenai.GPT3Dot5Turbo16K,
				Messages: []goOpenai.ChatCompletionMessage{
					{
						Role:    goOpenai.ChatMessageRoleUser,
						Content: task,
					},
				},
				Stream: true,
			}
			stream, err := m.client.CreateChatCompletionStream(ctx, req)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}

			tokenUsage.Input += llms.CountTokens(OPENAI_MODEL, task)

			totalOutput := 0

			for {
				completion, err := stream.Recv()

				if errors.Is(err, io.EOF) || err != nil {
					break
				}

				tokenUsage.Output = llms.CountTokens(OPENAI_MODEL, completion.Choices[0].Delta.Content)
				totalOutput += tokenUsage.Output

				result := Result{Result: completion.Choices[0].Delta.Content, TokenUsage: tokenUsage}

				chan_res <- result
			}

			if c != nil {
				(*c)(OPENAI_MODEL, tokenUsage.Input, totalOutput)
			}
			return
		}
		chan_res <- Result{
			Result:     "{\"error\":\"generation_failed\"}",
			TokenUsage: tokenUsage,
			Err: errors.New(
				"Generation failed after 5 retries",
			),
		}
		return
	}()

	return chan_res
}
