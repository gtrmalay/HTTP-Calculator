package unit

import (
	"testing"
	"time"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseToken(t *testing.T) {
	userID := 123
	token, err := auth.GenerateToken(userID)
	assert.NoError(t, err, "GenerateToken should not return an error")
	assert.NotEmpty(t, token, "Generated token should not be empty")

	claims, err := auth.ParseToken(token)
	assert.NoError(t, err, "ParseToken should not return an error")
	assert.Equal(t, userID, claims.UserID, "Parsed user ID should match")

	assert.True(t, claims.ExpiresAt.After(time.Now()), "Token expiration should be in the future")
	assert.True(t, claims.ExpiresAt.Before(time.Now().Add(25*time.Hour)), "Token expiration should be within 24 hours")
}

func TestParseTokenInvalid(t *testing.T) {
	invalidToken := "invalid.token.here"
	claims, err := auth.ParseToken(invalidToken)
	assert.Error(t, err, "ParseToken should return an error for invalid token")
	assert.Nil(t, claims, "Claims should be nil for invalid token")
}

func TestParseTokenWithBearer(t *testing.T) {
	userID := 456
	token, err := auth.GenerateToken(userID)
	assert.NoError(t, err)

	bearerToken := "Bearer " + token
	claims, err := auth.ParseToken(bearerToken)
	assert.NoError(t, err, "ParseToken should handle Bearer prefix")
	assert.Equal(t, userID, claims.UserID, "Parsed user ID should match with Bearer prefix")
}
