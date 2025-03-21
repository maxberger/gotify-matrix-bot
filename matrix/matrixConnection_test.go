package matrix_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	. "gotify_matrix_bot/matrix"

	"github.com/petergtz/pegomock/v4"
	"gotest.tools/v3/assert"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func setupMock(t *testing.T, downloadAllowList []*regexp.Regexp) (*MockMautrixClientType, *MatrixState) {
	mockClient := NewMockMautrixClientType(pegomock.WithT(t))
	state := &MatrixState{
		IsEncrypted:               false,
		MatrixContext:             context.Background(),
		MautrixClient:             mockClient,
		OlmMachine:                nil,
		DownloadFromHostAllowlist: downloadAllowList,
	}
	return mockClient, state
}

func setupHttpServer(contentType string) (*httptest.Server, string) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0, 1, 2, 3})
	}))
	return server, server.URL
}

func TestSendMessage(t *testing.T) {
	pegomock.RegisterMockTestingT(t)

	t.Run("Send unencrypted message", func(t *testing.T) {
		mockClient, state := setupMock(t, nil)

		roomID := "!room:example.com"
		message := "Test Message"
		content := format.RenderMarkdown(
			message,
			/*allowMarkdown = */ true,
			/*allowHTML = */ false)

		SendMessage(state, roomID, message)

		mockClient.VerifyWasCalledOnce().SendMessageEvent(
			pegomock.Any[context.Context](),
			pegomock.Eq(id.RoomID(roomID)),
			pegomock.Eq(event.EventMessage),
			pegomock.Eq(content))
	})

}

func TestUploadImages(t *testing.T) {
	pegomock.RegisterMockTestingT(t)

	t.Run("MessageWithoutLinksIsUnchanged", func(t *testing.T) {
		mockClient, state := setupMock(t, nil)

		message := "Test Message"
		result := UploadImages(state, message)

		assert.Equal(t, result, message)

		mockClient.VerifyWasCalled(pegomock.Never()).UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())
	})

	t.Run("Message with Link downloads link and changes URL", func(t *testing.T) {
		server, serverUrl := setupHttpServer("image/png")
		mockClient, state := setupMock(t, []*regexp.Regexp{
			regexp.MustCompile(".*"),
		})
		defer server.Close()

		pegomock.When(mockClient.UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())).
			ThenReturn(
				&mautrix.RespMediaUpload{
					ContentURI: id.MustParseContentURI("mxc://example.com/AQwafuaFswefuhsfAFAgsw"),
				},
				nil,
			)

		message := "Before\n![](" + serverUrl + "/image.png)\nAfter"
		result := UploadImages(state, message)
		assert.Equal(t, result, "Before\n![](mxc://example.com/AQwafuaFswefuhsfAFAgsw)\nAfter")

		mockClient.VerifyWasCalled(pegomock.Never()).SendMessageEvent(
			pegomock.Any[context.Context](),
			pegomock.Any[id.RoomID](),
			pegomock.Any[event.Type](),
			pegomock.Any[any]())
	})
	t.Run("Non-Images are passed through", func(t *testing.T) {
		server, serverUrl := setupHttpServer("text/plain")
		mockClient, state := setupMock(t, []*regexp.Regexp{
			regexp.MustCompile(".*"),
		})
		defer server.Close()

		message := "Before\n![](" + serverUrl + "/image.png)\nAfter"
		result := UploadImages(state, message)
		assert.Equal(t, result, message)

		mockClient.VerifyWasCalled(pegomock.Never()).UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())
	})

	t.Run("Image from non-allowlisted server is ignored", func(t *testing.T) {
		server, serverUrl := setupHttpServer("image/png")
		mockClient, state := setupMock(t, []*regexp.Regexp{
			regexp.MustCompile("some-other-server.com"),
		})
		defer server.Close()

		message := "Before\n![](" + serverUrl + "/image.png)\nAfter"
		result := UploadImages(state, message)
		assert.Equal(t, result, message)

		mockClient.VerifyWasCalled(pegomock.Never()).UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())
	})

	t.Run("Image is ignored if allowlist is empty", func(t *testing.T) {
		server, serverUrl := setupHttpServer("image/png")
		mockClient, state := setupMock(t, []*regexp.Regexp{})
		defer server.Close()

		message := "Before\n![](" + serverUrl + "/image.png)\nAfter"
		result := UploadImages(state, message)
		assert.Equal(t, result, message)

		mockClient.VerifyWasCalled(pegomock.Never()).UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())
	})

	t.Run("Image is ignored if allowlist is nil", func(t *testing.T) {
		server, serverUrl := setupHttpServer("image/png")
		mockClient, state := setupMock(t, nil)
		defer server.Close()

		message := "Before\n![](" + serverUrl + "/image.png)\nAfter"
		result := UploadImages(state, message)
		assert.Equal(t, result, message)

		mockClient.VerifyWasCalled(pegomock.Never()).UploadMedia(
			pegomock.Any[context.Context](),
			pegomock.Any[mautrix.ReqUploadMedia]())
	})

}
