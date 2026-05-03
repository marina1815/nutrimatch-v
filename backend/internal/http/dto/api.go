package dto

import "time"

type ResponseMeta struct {
	RequestID string    `json:"requestId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type SuccessResponse[T any] struct {
	Data T            `json:"data"`
	Meta ResponseMeta `json:"meta"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody    `json:"error"`
	Meta  ResponseMeta `json:"meta"`
}
