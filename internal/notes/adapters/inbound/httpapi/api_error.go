package httpapi

import "net/http"

// APIError is the one error body this API returns. Anything in this adapter can
// return one to stop a request. The core can't import it, so translate converts
// the core's errors.
type APIError struct {
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Category  string       `json:"category"`
	Code      string       `json:"code"`
	Detail    string       `json:"detail"`
	RequestID string       `json:"requestId,omitempty"`
	Errors    []FieldError `json:"errors,omitempty"`
}

// NewAPIError builds a technical error.
func NewAPIError(status int, code, detail string) APIError {
	return APIError{Title: http.StatusText(status), Status: status, Category: categoryTechnical, Code: code, Detail: detail}
}

// NewBusinessError builds an error for a broken business rule.
func NewBusinessError(status int, code, detail string) APIError {
	apiError := NewAPIError(status, code, detail)
	apiError.Category = categoryBusiness
	return apiError
}

func (apiError APIError) Error() string { return apiError.Detail }

func (apiError APIError) withFieldErrors(fieldErrors ...FieldError) APIError {
	apiError.Errors = fieldErrors
	return apiError
}
