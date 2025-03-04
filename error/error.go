package customerror

import "fmt"

type WebError struct {
	StatusCode int
	Message    string
}

type StandardError struct {
	Message string
}

// Implement the error interface for WebError
func (e *WebError) Error() string {
	return fmt.Sprintf("Error: status_code %d, message: %s", e.StatusCode, e.Message)
}

// Implement the error interface for StandardError
func (e *StandardError) Error() string {
	return fmt.Sprintf("Error message: %s", e.Message)
}
