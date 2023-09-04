package middlewares

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"

	posthog "github.com/polyfact/api/posthog"
	"github.com/polyfact/api/utils"
)

var isDevelopment = os.Getenv("APP_MODE") == "development"

func RecoverFromPanic(w http.ResponseWriter, r *http.Request) {
	record := r.Context().Value("recordEvent").(utils.RecordFunc)
	if rec := recover(); rec != nil {
		errorMessage := getErrorMessage(rec)

		utils.RespondError(w, record, "unknown_error", errorMessage)
	}
}

func getErrorMessage(rec interface{}) string {
	if isDevelopment {
		switch v := rec.(type) {
		case error:
			return v.Error()
		case string:
			return v
		}
	}

	// For prod or for unhandled types
	return "Internal Server Error"
}

func AddRecord(r *http.Request) {
	var recordEventRequest utils.RecordRequestFunc
	recordEventRequest = func(request string, response string, userID string, props ...utils.KeyValue) {
		properties := make(map[string]string)
		properties["path"] = string(r.URL.Path)
		properties["requestBody"] = request
		properties["responseBody"] = response
		for _, element := range props {
			properties[element.Key] = element.Value
		}
		posthog.Event("API Request", userID, properties)
	}

	buf, _ := ioutil.ReadAll(r.Body)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))

	r.Body = rdr1

	var recordEventWithUserID utils.RecordWithUserIDFunc
	recordEventWithUserID = func(response string, userID string, props ...utils.KeyValue) {
		recordEventRequest(string(buf), response, userID, props...)
	}

	var recordEvent utils.RecordFunc
	recordEvent = func(response string, props ...utils.KeyValue) {
		recordEventWithUserID(response, "", props...)
	}

	newCtx := context.WithValue(r.Context(), "recordEvent", recordEvent)
	newCtx = context.WithValue(newCtx, "recordEventRequest", recordEventRequest)
	newCtx = context.WithValue(newCtx, "recordEventWithUserID", recordEventWithUserID)

	*r = *r.WithContext(newCtx)
}