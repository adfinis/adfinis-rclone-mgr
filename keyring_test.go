package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func TestCredentialGetAndSet(t *testing.T) {
	keyring.MockInit()

	clientID := "test_client_id"
	clientSecret := "test_client_secret"

	err := setCredentials(clientID, clientSecret)
	assert.NoError(t, err)

	retrievedClientID, retrievedClientSecret, err := getCredentials()
	assert.NoError(t, err)
	assert.Equal(t, clientID, retrievedClientID)
	assert.Equal(t, clientSecret, retrievedClientSecret)
}
