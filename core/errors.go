package core

import "fmt"

type HTTPError struct {
	Status  int
	Message string
}

func (err *HTTPError) Error() string {
	return err.Message
}

func NewHTTPError(status int, message string) *HTTPError {
	return &HTTPError{
		Status:  status,
		Message: message,
	}
}

func Errorf(status int, format string, args ...any) *HTTPError {
	return &HTTPError{
		Status:  status,
		Message: fmt.Sprintf(format, args...),
	}
}
