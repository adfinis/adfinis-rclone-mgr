package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func TestHttpHandlerRoot(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("GET", "/", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rw.Code)
	}
}

func TestHttpHandlerLoginMissingFields(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerAuthInvalidState(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("GET", "/auth?state=invalid", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerGenerateMissingToken(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/generate", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerGenerateMissingOidc(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/generate", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: base64.StdEncoding.EncodeToString([]byte("dummy"))})
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

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

func TestSanitizeDriveName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"My Drive", "My_Drive"},
		{"My/Drive", "My_Drive"},
		{"My\\Drive", "My_Drive"},
		{"My:Drive", "My_Drive"},
		{"My?Drive", "My_Drive"},
		{"My*Drive", "My_Drive"},
		{"My\"Drive", "My_Drive"},
		{"My<Drive", "My_Drive"},
		{"My>Drive", "My_Drive"},
		{"My|Drive", "My_Drive"},
		{"My&Drive", "My_Drive"},
	} {
		result := sanitizeDriveName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestUnitNameToDriveName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"rclone@mydrive.service", "mydrive"},
		{"rclone@my-drive-1.service", "my-drive-1"},
		{"rclone@my_drive_xyz.service", "my_drive_xyz"},
	} {
		result := unitNameToDriveName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestDriveNameToUnitName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"mydrive", "rclone@mydrive.service"},
		{"my-drive-1", "rclone@my-drive-1.service"},
		{"my_drive_xyz", "rclone@my_drive_xyz.service"},
	} {
		result := driveNameToUnitName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestStatusesToServiceStatuses(t *testing.T) {
	for _, test := range []struct {
		input    []dbus.UnitStatus
		expected []serviceStatus
	}{
		{
			input: []dbus.UnitStatus{
				{
					Name:        "rclone@test.service",
					Description: "Test Service",
					ActiveState: "active",
				},
				{
					Name:        "rclone@my_drive.service",
					Description: "My Drive Service",
					ActiveState: "inactive",
				},
			},
			expected: []serviceStatus{
				{
					Name:      "test",
					Status:    "active",
					MountPath: path.Join(xdg.Home, "google", "test"),
				},
				{
					Name:      "my_drive",
					Status:    "inactive",
					MountPath: path.Join(xdg.Home, "google", "my_drive"),
				},
			},
		},
	} {
		result := statusesToServiceStatuses(test.input)
		assert.Equal(t, test.expected, result)
	}

}
