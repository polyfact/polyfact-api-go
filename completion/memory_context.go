package completion

import (
	"context"

	"github.com/polyfire/api/db"
	"github.com/polyfire/api/memory"
	"github.com/polyfire/api/utils"
)

func parseMemoryIdArray(memoryId interface{}) []string {
	var memoryIdArray []string

	if str, ok := memoryId.(string); ok {
		memoryIdArray = append(memoryIdArray, str)
	} else if array, ok := memoryId.([]interface{}); ok {
		for _, item := range array {
			if str, ok := item.(string); ok {
				memoryIdArray = append(memoryIdArray, str)
			}
		}
	}
	return memoryIdArray
}

type MemoryProcessResult struct {
	Resources         []db.MatchResult
	ContextCompletion string
}

func getMemory(ctx context.Context, user_id string, memoryId interface{}, task string) (*MemoryProcessResult, error) {
	memoryIdArray := parseMemoryIdArray(memoryId)
	if memoryIdArray == nil {
		return nil, nil
	}

	results, err := memory.Embedder(ctx, user_id, memoryIdArray, task)
	if err != nil {
		return nil, InternalServerError
	}

	context_completion, err := utils.FillContext(results)
	if err != nil {
		return nil, InternalServerError
	}

	response := &MemoryProcessResult{
		ContextCompletion: context_completion,
		Resources:         results,
	}

	return response, nil
}
