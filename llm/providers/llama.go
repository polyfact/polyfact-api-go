package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type LLaMaInputBody struct {
	Prompt string `json:"prompt"`
}

type LLaMaProvider struct{}

func (m LLaMaProvider) Generate(task string, c *func(string, int, int), opts *ProviderOptions) chan Result {
	chan_res := make(chan Result)

	go func() {
		defer close(chan_res)
		tokenUsage := TokenUsage{Input: 0, Output: 0}
		body := LLaMaInputBody{Prompt: task}
		fmt.Println(body)
		input, err := json.Marshal(body)
		tokenUsage.Input += llms.CountTokens("gpt-2", task)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		reqBody := string(input)
		fmt.Println(reqBody)
		resp, err := http.Post(os.Getenv("LLAMA_URL"), "application/json", strings.NewReader(reqBody))
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		defer resp.Body.Close()
		var p []byte = make([]byte, 128)
		for {
			nb, err := resp.Body.Read(p)
			if errors.Is(err, io.EOF) || err != nil {
				break
			}
			tokenUsage.Output = llms.CountTokens("gpt-2", string(p[:nb]))
			chan_res <- Result{Result: string(p[:nb]), TokenUsage: tokenUsage}
		}
	}()

	return chan_res
}