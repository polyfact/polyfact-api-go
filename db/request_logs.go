package db

type Kind string

const (
	Completion Kind = "completion"
	Embed      Kind = "embedding"
)

type RequestLog struct {
	UserID           string `json:"user_id"`
	ModelName        string `json:"model_name"`
	InputTokenCount  *int   `json:"input_token_count"`
	OutputTokenCount *int   `json:"output_token_count"`
	Kind             Kind   `json:"kind"`
}

func toRef[T interface{}](n T) *T {
	return &n
}

func LogRequests(
	user_id string,
	model_name string,
	input_token_count int,
	output_token_count int,
	kind Kind,
) {
	supabase, err := CreateClient()
	if err != nil {
		panic(err)
	}

	if kind == "" {
		kind = "completion"
	}

	row := RequestLog{
		UserID:           user_id,
		ModelName:        model_name,
		Kind:             kind,
		InputTokenCount:  toRef(input_token_count),
		OutputTokenCount: toRef(output_token_count),
	}

	_, _, err = supabase.From("request_logs").Insert(row, false, "", "", "exact").Execute()
	if err != nil {
		panic(err)
	}
}