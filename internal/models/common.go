package models

type Pagination struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Code    int    `json:"code,omitempty"`
}
