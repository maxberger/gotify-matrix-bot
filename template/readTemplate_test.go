package template

import (
	"testing"
)

func TestGetFormattedMessageString(t *testing.T) {

	// Test cases
	testCases := []struct {
		name          string
		message       []byte
		expected      string
		expectedError bool
	}{
		{
			name:          "Valid message",
			message:       []byte(`{"title": "Test Title", "message": "Test Message"}`),
			expected:      "### Test Title\n\nTest Message\n",
			expectedError: false,
		},
		{
			name:          "Invalid JSON",
			message:       []byte(`{"title": "Test Title", "message": "Test Message"`),
			expected:      "Could not parse message from: {\"title\": \"Test Title\", \"message\": \"Test Message\"",
			expectedError: true,
		},
		{
			name:          "Empty message",
			message:       []byte(`{"title": "", "message": ""}`),
			expected:      "### \n\n\n",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetFormattedMessageString(tc.message)
			if result != tc.expected {
				t.Errorf("GetFormattedMessageString() = %q, want %q", result, tc.expected)
			}
		})
	}
}
