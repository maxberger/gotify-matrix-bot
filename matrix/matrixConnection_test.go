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

func TestRewriteInRoomVerificationEventID(t *testing.T) {
	t.Run("Verification ready event with m.reference relation gets ID rewritten", func(t *testing.T) {
		originalRequestID := id.EventID("$request-event-id")
		readyEventID := id.EventID("$ready-event-id")

		evt := &event.Event{
			ID:   readyEventID,
			Type: event.InRoomVerificationReady,
			Content: event.Content{
				Parsed: &event.VerificationReadyEventContent{
					InRoomVerificationEvent: event.InRoomVerificationEvent{
						RelatesTo: &event.RelatesTo{
							Type:    event.RelReference,
							EventID: originalRequestID,
						},
					},
				},
			},
		}

		RewriteInRoomVerificationEventID(evt)

		assert.Equal(t, evt.ID, originalRequestID)
	})

	t.Run("Event without relation does not get ID rewritten", func(t *testing.T) {
		readyEventID := id.EventID("$ready-event-id")

		evt := &event.Event{
			ID:   readyEventID,
			Type: event.InRoomVerificationReady,
			Content: event.Content{
				Parsed: &event.VerificationReadyEventContent{},
			},
		}

		RewriteInRoomVerificationEventID(evt)

		assert.Equal(t, evt.ID, readyEventID)
	})
}

func TestFilterOwnVerificationEvent(t *testing.T) {
	ownUserID := id.UserID("@bot:example.com")
	otherUserID := id.UserID("@user:example.com")

	t.Run("Verification ready event sent by ourselves is filtered", func(t *testing.T) {
		evt := &event.Event{
			Sender: ownUserID,
			Type:   event.InRoomVerificationReady,
		}

		filtered := FilterOwnVerificationEvent(evt, ownUserID)

		assert.Equal(t, filtered, true)
		assert.Equal(t, evt.Type.Type, "ignored.verification")
	})

	t.Run("Verification ready event sent by others is not filtered", func(t *testing.T) {
		evt := &event.Event{
			Sender: otherUserID,
			Type:   event.InRoomVerificationReady,
		}

		filtered := FilterOwnVerificationEvent(evt, ownUserID)

		assert.Equal(t, filtered, false)
		assert.Equal(t, evt.Type, event.InRoomVerificationReady)
	})

	t.Run("Normal message sent by ourselves is not filtered", func(t *testing.T) {
		evt := &event.Event{
			Sender: ownUserID,
			Type:   event.EventMessage,
		}

		filtered := FilterOwnVerificationEvent(evt, ownUserID)

		assert.Equal(t, filtered, false)
		assert.Equal(t, evt.Type, event.EventMessage)
	})
}
