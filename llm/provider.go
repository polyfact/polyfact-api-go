package llm

import (
	"context"
	"errors"
	"log"

	"github.com/polyfire/api/codegen"
	database "github.com/polyfire/api/db"
	"github.com/polyfire/api/llm/providers"
	"github.com/polyfire/api/llm/providers/options"
	"github.com/polyfire/api/utils"
	"github.com/tmc/langchaingo/llms/cohere"
)

var ErrUnknownModel = errors.New("Unknown model")

type Provider interface {
	Name() string
	ProviderModel() (string, string)
	Generate(
		prompt string,
		c options.ProviderCallback,
		opts *options.ProviderOptions,
	) chan options.Result
	DoesFollowRateLimit() bool
}

func getAvailableModels(model string) (string, string) {
	switch model {
	case "cheap":
		return "llama", "llama2"
	case "regular":
		return "openai", "gpt-3.5-turbo"
	case "best":
		return "openai", "gpt-4"
	case "uncensored":
		return "replicate", "wizard-mega-13b-awq"
	case "gpt-3.5-turbo":
		return "openai", "gpt-3.5-turbo"
	case "gpt-3.5-turbo-16k":
		return "openai", "gpt-3.5-turbo"
	case "gpt-4":
		return "openai", "gpt-4"
	case "gpt-4-32k":
		return "openai", "gpt-4-32k"
	case "gpt-4o":
		return "openai", "gpt-4o"
	case "gpt-4o-mini":
		return "openai", "gpt-4o-mini"
	case "gpt-4-turbo":
		return "openai", "gpt-4-turbo"
	case "cohere":
		return "cohere", "cohere_command"
	case "llama-2-70b-chat":
		return "replicate", "llama-2-70b-chat"
	case "replit-code-v1-3b":
		return "replicate", "replit-code-v1-3b"
	case "wizard-mega-13b-awq":
		return "replicate", "wizard-mega-13b-awq"
	case "airoboros-llama-2-70b":
		return "replicate", "airoboros-llama-2-70b"
	case "":
		return "openai", "gpt-3.5-turbo"
	}
	if codegen.IsOpenRouterModel(model) {
		return "openrouter", model
	}
	return "", ""
}

func getModelWithAliases(
	ctx context.Context,
	modelAlias string,
	projectID string,
) (string, string) {
	db := ctx.Value(utils.ContextKeyDB).(database.Database)
	provider, model := getAvailableModels(modelAlias)

	if model == "" {
		newModel, err := db.GetModelByAliasAndProjectID(modelAlias, projectID, "completion")
		if err != nil {
			return "", ""
		}

		model = newModel.Model
		provider = newModel.Provider
	}

	return provider, model
}

func NewProvider(ctx context.Context, modelInput string) (Provider, error) {
	projectID, _ := ctx.Value(utils.ContextKeyProjectID).(string)
	log.Println("[INFO] Project ID: ", projectID)

	provider, model := getModelWithAliases(ctx, modelInput, projectID)

	log.Println("[INFO] Provider: ", provider)

	switch provider {
	case "openai":
		log.Println("[INFO] Using OpenAI")
		llm := providers.NewOpenAIStreamProvider(ctx, model)

		return llm, nil
	case "cohere":
		log.Println("[INFO] Using Cohere")
		llm, err := cohere.New()
		if err != nil {
			return nil, err
		}
		return providers.LangchainProvider{Model: llm, ModelName: "cohere_command"}, nil
	case "llama":
		return providers.LLaMaProvider{
			Model: model,
		}, nil
	case "replicate":
		log.Println("[INFO] Using Replicate")
		llm := providers.NewReplicateProvider(ctx, model)
		return llm, nil
	case "openrouter":
		log.Println("[INFO] Using OpenRouter")
		llm := providers.NewOpenRouterProvider(ctx, model)

		return llm, nil
	default:
		return nil, ErrUnknownModel
	}
}
