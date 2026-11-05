package httpapi

// FieldError is one problem found in a request.
type FieldError struct {
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}
