package api

import (
	"encoding/json"
	"net/http"
)

type LoginParams struct {
	Email    string
	Password string
}

type LoginResponse struct {
	Code     int    `json:"code"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Message  string `json:"message"`
	Token    string `json:"token"`
	Verified bool   `json:"verified"`
}

type VerifyOtpParams struct {
	UserID string `json:"user_id"`
	OTP    string `json:"otp"`
}

type VerifyOtpResponse struct {
	Code     int    `json:"code"`
	Verified bool   `json:"verified"`
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
}

type Error struct {
	//error code
	Code int

	//Error Message
	Message string
}

func writeError(w http.ResponseWriter, message string, code int) {
	resp := Error{
		Code:    code,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(resp)
}

var (
	RequestErrorHandler = func(w http.ResponseWriter, err error) {
		writeError(w, err.Error(), http.StatusBadRequest)
	}
	InternalErrorHandler = func(w http.ResponseWriter) {
		writeError(w, "An Unexpected error occurred.", http.StatusInternalServerError)
	}

	UnAuthorizedError = func(w http.ResponseWriter) {
		writeError(w, "Invalid Username or Token", http.StatusUnauthorized)
	}

	DuplicateError = func(w http.ResponseWriter) {
		writeError(w, "This User exists", http.StatusConflict)
	}
)
