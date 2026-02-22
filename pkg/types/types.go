package types

import "fmt"

type ErrorCode string

const (
	ErrConfigNotFound     ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrProviderAuthFailed ErrorCode = "PROVIDER_AUTH_FAILED"
	ErrProviderRateLimit  ErrorCode = "PROVIDER_RATE_LIMIT"
	ErrProviderError      ErrorCode = "PROVIDER_ERROR"
	ErrKeyNotFound        ErrorCode = "KEY_NOT_FOUND"
	ErrFileAccessDenied   ErrorCode = "FILE_ACCESS_DENIED"
	ErrFileParseError     ErrorCode = "FILE_PARSE_ERROR"
	ErrRotationFailed     ErrorCode = "ROTATION_FAILED"
	ErrRollbackFailed     ErrorCode = "ROLLBACK_FAILED"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewError(code ErrorCode, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}
