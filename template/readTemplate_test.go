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
		{
			name: "MarkdownResult = title + message",
			message: []byte(`{
            "id": 733,
            "appid": 4,
            "message": "This is a test message with Markdown\n\r![](https://some.link.com/image.png)\n",
            "title": "Test Notification",
            "priority": 5,
            "extras": {
                "client::display": {
                    "contentType": "text/markdown"
                }
            },
            "date": "2025-03-01T00:01:02.223071187+01:00"
        }`),
			expected:      "# Test Notification\n\n" + "This is a test message with Markdown\n\r![](https://some.link.com/image.png)\n",
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
