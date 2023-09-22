package completion

import (
	"github.com/polyfact/api/db"
)

func getSystemPrompt(system_prompt_id *string, system_prompt *string) (string, error) {
	var result string = ""

	if system_prompt != nil && len(*system_prompt) > 0 {
		result = *system_prompt
	}

	if system_prompt_id != nil && len(*system_prompt_id) > 0 {
		p, err := db.GetPromptById(*system_prompt_id)
		if err != nil || p == nil {
			return "", NotFound
		}

		_, err = db.UpdatePromptUse(*system_prompt_id, db.PromptUse{Use: p.Use + 1})
		if err != nil {
			return "", NotFound
		}
		result = p.Prompt
	}

	return result, nil
}