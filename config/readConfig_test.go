package config

import (
	"regexp"
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
		{
			name: "valid data with allowList",
			config: []byte(`
downloadFromHostAllowlist: 
  - ".*\\.yourdomain\\.com"
  - ".*\\.trusteddomain\\.com"
`),
			expected: &Config{
				Gotify: GotifyType{
					URL: "wss://",
				},
				Logging: LoggingType{
					Level: "info",
				},
				DownloadFromHostAllowlist: []string{`.*\.yourdomain\.com`, `.*\.trusteddomain\.com`},
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

func TestDownloadAllowListAsRegexps(t *testing.T) {
	testCases := []struct {
		name          string
		config        *Config
		expected      *[]regexp.Regexp
		expectedError string
	}{
		{
			name: "Valid allow list",
			config: &Config{
				DownloadFromHostAllowlist: []string{`www\.host1\.com`, `www\.host2\.com`},
			},
			expected: &[]regexp.Regexp{
				*regexp.MustCompile(`www\.host1\.com`),
				*regexp.MustCompile(`www\.host2\.com`),
			},
			expectedError: "",
		},
		{
			name: "Invalid Regexp",
			config: &Config{
				DownloadFromHostAllowlist: []string{`$$$^^(`},
			},
			expected:      nil,
			expectedError: "error parsing regexp",
		},
		{
			name: "Empty list works",
			config: &Config{
				DownloadFromHostAllowlist: []string{},
			},
			expected:      &[]regexp.Regexp{},
			expectedError: "",
		},
		{
			name: "Nil list works",
			config: &Config{
				DownloadFromHostAllowlist: nil,
			},
			expected:      &[]regexp.Regexp{},
			expectedError: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DownloadAllowListAsRegexps(tc.config)
			if len(tc.expectedError) == 0 {
				assert.Equal(t, len(result), len(*tc.expected))
				for i, r := range result {
					assert.Equal(t, r.String(), (*tc.expected)[i].String())
				}
			} else {
				assert.ErrorContains(t, err, tc.expectedError)
				assert.Assert(t, result == nil)
			}
		})
	}
}
