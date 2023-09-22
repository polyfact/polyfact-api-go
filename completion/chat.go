package completion

import (
	"encoding/json"
	"log"
	"net/http"

	router "github.com/julienschmidt/httprouter"
	"github.com/polyfact/api/db"
	"github.com/polyfact/api/llm/providers"
	"github.com/polyfact/api/utils"
)

func FormatPrompt(chatHistory []db.ChatMessage, userPrompt string) string {
	res := ""

	for i := len(chatHistory) - 1; i >= 0; i-- {
		if chatHistory[i].IsUserMessage {
			res += "\nHuman: " + chatHistory[i].Content
		} else {
			res += "\nAI: " + chatHistory[i].Content
		}
	}

	res += "\nHuman: " + userPrompt + "\nAI: "

	return res
}

func CreateChat(w http.ResponseWriter, r *http.Request, _ router.Params) {
	user_id := r.Context().Value(utils.ContextKeyUserID).(string)
	record := r.Context().Value(utils.ContextKeyRecordEvent).(utils.RecordFunc)

	var requestBody struct {
		SystemPrompt   *string `json:"system_prompt"`
		SystemPromptId *string `json:"system_prompt_id"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestBody); err != nil {
		utils.RespondError(w, record, "decode_error")
		return
	}

	if r.Method != "POST" {
		utils.RespondError(w, record, "only_post_method_allowed")
		return
	}

	chat, err := db.CreateChat(user_id, requestBody.SystemPrompt, requestBody.SystemPromptId)
	if err != nil {
		log.Printf("Error creating chat for user %s : %v", user_id, err)
		utils.RespondError(w, record, "error_create_chat", err.Error())
		return
	}

	response, _ := json.Marshal(&chat)
	record(string(response))

	_ = json.NewEncoder(w).Encode(chat)
}

func GetChatHistory(w http.ResponseWriter, r *http.Request, ps router.Params) {
	id := ps.ByName("id")
	user_id := r.Context().Value(utils.ContextKeyUserID).(string)
	record := r.Context().Value(utils.ContextKeyRecordEvent).(utils.RecordFunc)

	messages, err := db.GetChatMessages(user_id, id)
	if err != nil {
		utils.RespondError(w, record, "error_chat_history")
		return
	}

	response, _ := json.Marshal(&messages)
	record(string(response))

	_ = json.NewEncoder(w).Encode(messages)
}

func chatContext(
	user_id string,
	task string,
	chatId string,
	system_prompt *string,
	callback providers.ProviderCallback,
	opts *providers.ProviderOptions,
) (string, error) {
	chat, err := db.GetChatById(chatId)
	if err != nil {
		return "", InternalServerError
	}

	if chat == nil || chat.UserID != user_id {
		return "", NotFound
	}

	allHistory, err := db.GetChatMessages(user_id, chatId)
	if err != nil {
		return "", InternalServerError
	}

	chatHistory := utils.CutChatHistory(allHistory, 1000)

	if (system_prompt == nil || *system_prompt == "") && chat.SystemPrompt != nil {
		*system_prompt = *(chat.SystemPrompt)
	}

	prompt := FormatPrompt(chatHistory, task)

	err = db.AddChatMessage(chat.ID, true, task)
	if err != nil {
		return "", InternalServerError
	}

	old_callback := *callback
	*callback = func(provider_name string, model_name string, input_count int, output_count int, completion string) {
		if old_callback != nil {
			old_callback(provider_name, model_name, input_count, output_count, completion)
		}
		_ = db.AddChatMessage(chat.ID, false, completion)
	}

	opts.StopWords = &[]string{"AI:", "Human:"}

	return prompt, nil
}
