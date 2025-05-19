package main

import (
	"github.com/zalando/go-keyring"
)

const keyringService = "adfinis-rclone-mgr"

func getCredentials() (clientID string, clientSecret string, err error) {
	clientID, err = keyring.Get(keyringService, "client_id")
	if err != nil {
		return
	}
	clientSecret, err = keyring.Get(keyringService, "client_secret")
	if err != nil {
		return
	}
	return
}

func setCredentials(clientID string, clientSecret string) error {
	if err := keyring.Set(keyringService, "client_id", clientID); err != nil {
		return err
	}
	if err := keyring.Set(keyringService, "client_secret", clientSecret); err != nil {
		return err
	}
	return nil
}
