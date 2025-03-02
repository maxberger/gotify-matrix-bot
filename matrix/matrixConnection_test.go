package matrix_test

import (
	"context"
	"testing"

	. "gotify_matrix_bot/matrix"

	"github.com/petergtz/pegomock/v4"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func TestSendMessage(t *testing.T) {
	pegomock.RegisterMockTestingT(t)

	t.Run("Send unencrypted message", func(t *testing.T) {
		mockClient := NewMockMautrixClient(pegomock.WithT(t))
		state := &MatrixState{
			IsEncrypted:   false,
			MatrixContext: context.Background(),
			MautrixClient: mockClient,
			OlmMachine:    nil,
		}
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
