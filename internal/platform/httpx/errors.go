package httpx

import "net/http"

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, code, message string, details ...any) {
	body := errorBody{Code: code, Message: message}
	if len(details) > 0 {
		body.Details = details[0]
	}
	WriteJSON(w, status, errorEnvelope{Error: body})
}
