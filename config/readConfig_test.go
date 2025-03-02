package config

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseConfig(t *testing.T) {

	// Test cases
	testCases := []struct {
		name          string
		config        []byte
		expected      *Config
		expectedError bool
	}{
		{
			name: "Valid config without domain",
			config: []byte(`
gotify:
  url: https://gotify.example.com
  apiToken: testToken
matrix:
  homeserverURL: https://matrix.example.com
  username: testuser
  token: testToken
  roomID: "!roomid"
  encrypted: true
logging:
  level: debug
  format: color
`),
			expected: &Config{
				Gotify: GotifyType{
					URL:      "wss://gotify.example.com",
					ApiToken: "testToken"},
				Matrix: MatrixType{
					HomeServerURL: "https://matrix.example.com",
					Username:      "testuser",
					Token:         "testToken",
					RoomID:        "!roomid",
					MatrixDomain:  "matrix.example.com",
					Encrypted:     true,
				},
				Logging: LoggingType{
					Level:  "debug",
					Format: "color",
				},
			},
			expectedError: false,
		},
		{
			name: "Valid config with domain",
			config: []byte(`
gotify:
  url: https://gotify.example.com
  apiToken: testToken
matrix:
  homeserverURL: https://matrix.example.com
  matrixDomain: example.com
  username: testuser
  token: testToken
  roomID: "!roomid"
  encrypted: true
debug: true
`),
			expected: &Config{
				Gotify: GotifyType{
					URL:      "wss://gotify.example.com",
					ApiToken: "testToken"},
				Matrix: MatrixType{
					HomeServerURL: "https://matrix.example.com",
					Username:      "testuser",
					Token:         "testToken",
					RoomID:        "!roomid",
					MatrixDomain:  "example.com",
					Encrypted:     true,
				},
				Debug: true,
				Logging: LoggingType{
					Level: "debug",
				},
			},
			expectedError: false,
		},
		{
			name: "Debug sets level if unset",
			config: []byte(`
debug: true
`),
			expected: &Config{
				Gotify: GotifyType{
					URL: "wss://",
				},
				Debug: true,
				Logging: LoggingType{
					Level: "debug",
				},
			},
			expectedError: false,
		},
		{
			name: "Log level defaults to info",
			config: []byte(`
`),
			expected: &Config{
				Gotify: GotifyType{
					URL: "wss://",
				},
				Logging: LoggingType{
					Level: "info",
				},
			},
			expectedError: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseConfig(tc.config)
			assert.DeepEqual(t, result, tc.expected)
		})
	}
}
