package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "error without cause",
			err: &AppError{
				Code:    ErrConfigNotFound,
				Message: "config file missing",
			},
			expected: "CONFIG_NOT_FOUND: config file missing",
		},
		{
			name: "error with cause",
			err: &AppError{
				Code:    ErrProviderAuthFailed,
				Message: "invalid token",
				Cause:   errors.New("401 unauthorized"),
			},
			expected: "PROVIDER_AUTH_FAILED: invalid token (401 unauthorized)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &AppError{
		Code:    ErrRotationFailed,
		Message: "could not rotate",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestNewError(t *testing.T) {
	cause := errors.New("test cause")
	err := NewError(ErrKeyNotFound, "API_KEY not found", cause)

	assert.Equal(t, ErrKeyNotFound, err.Code)
	assert.Equal(t, "API_KEY not found", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrConfigNotFound,
		ErrConfigInvalid,
		ErrProviderAuthFailed,
		ErrProviderRateLimit,
		ErrProviderError,
		ErrKeyNotFound,
		ErrFileAccessDenied,
		ErrFileParseError,
		ErrRotationFailed,
		ErrRollbackFailed,
	}

	// Verify all codes are unique
	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate error code: %s", code)
		seen[code] = true
	}
}
