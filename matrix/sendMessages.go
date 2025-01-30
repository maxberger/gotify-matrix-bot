// Code by tulir from https://mau.dev/-/snippets/6

package matrix

import (
	"context"
	"gotify_matrix_bot/config"
	"log"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func SendEncrypted(ctx context.Context, mach *crypto.OlmMachine, cli *mautrix.Client, roomID id.RoomID, text string) (err error) {

	if config.Configuration.Debug {
		log.Println("Sending new encrypted message")
	}

	content := format.RenderMarkdown(text, true, true)
	encrypted, err := mach.EncryptMegolmEvent(ctx, roomID, event.EventMessage, content)
	// These three errors mean we have to make a new Megolm session
	if err == crypto.SessionExpired || err == crypto.SessionNotShared || err == crypto.NoGroupSession {
		err = mach.ShareGroupSession(ctx, roomID, getUserIDs(ctx, cli, roomID))
		if err != nil {
			return err
		}
		encrypted, err = mach.EncryptMegolmEvent(ctx, roomID, event.EventMessage, content)
	}
	if err != nil {
		return err
	}
	_, err = cli.SendMessageEvent(ctx, roomID, event.EventEncrypted, encrypted)
	if err != nil {
		return err
	}

	return nil
}

func SendUnencrypted(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, text string) (err error) {

	if config.Configuration.Debug {
		log.Println("Sending new unencrypted message")
	}

	_, err = cli.SendMessageEvent(ctx, roomID, event.EventMessage, format.RenderMarkdown(text, true, true))

	if err != nil {
		return err
	}

	return nil

}
