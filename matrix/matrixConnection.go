package matrix

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"

	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

type MautrixClientType interface {
	SendMessageEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, content interface{}, extra ...mautrix.ReqSendEvent) (*mautrix.RespSendEvent, error)
	JoinedMembers(ctx context.Context, roomID id.RoomID) (resp *mautrix.RespJoinedMembers, err error)
	Sync() error
}

type MatrixState struct {
	IsEncrypted   bool
	MatrixContext context.Context
	MautrixClient MautrixClientType
	OlmMachine    *crypto.OlmMachine
}

func Connect(
	homeServerURL string,
	username string,
	domain string,
	token string,
	encrypted bool,
) *MatrixState {
	var mach *crypto.OlmMachine = nil
	ctx := context.Background()
	cli, err := mautrix.NewClient(
		homeServerURL,
		id.UserID("@"+username+":"+domain),
		token)
	if err != nil {
		panic(err)
	}

	// DeviceID is needed for some older clients, e.g. some versions Element
	cli.DeviceID = id.DeviceID(strings.ToUpper(random.String(10)))

	if encrypted {
		cryptoStore := crypto.NewMemoryStore(nil)

		mach = crypto.NewOlmMachine(cli, &log.Logger, cryptoStore, &FakeStateStore{})
		// Load data from the crypto store
		err = mach.Load(ctx)
		if err != nil {
			panic(err)
		}

		// Hook up the OlmMachine into the Matrix client so it receives e2ee keys and other such things.
		syncer := cli.Syncer.(*mautrix.DefaultSyncer)
		syncer.OnSync(func(ctx context.Context, resp *mautrix.RespSync, since string) bool {
			mach.ProcessSyncResponse(ctx, resp, since)
			return true
		})
		syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
			mach.HandleMemberEvent(ctx, evt)
		})

	}

	// Start long polling in the background
	go func() {
		err = cli.Sync()
		if err != nil {
			panic(err)
		}
	}()

	log.Info().Msgf("Connected to Matrix. Encryption active: %t", encrypted)

	return &MatrixState{
		IsEncrypted:   encrypted,
		MatrixContext: ctx,
		MautrixClient: cli,
		OlmMachine:    mach,
	}
}

func SendMessage(state *MatrixState, roomID string, message string) {
	matrixRoomId := id.RoomID(roomID)

	log.Info().Msg("Sending new message")
	log.Debug().Msgf("Message: %s", message)
	log.Debug().Msgf("Room ID: %s", matrixRoomId)

	content := format.RenderMarkdown(
		message,
		/*allowMarkdown = */ true,
		/*allowHTML = */ false)

	var eventType event.Type
	var eventContent any

	if state.IsEncrypted {
		encryptedContent, err := state.OlmMachine.EncryptMegolmEvent(
			state.MatrixContext,
			matrixRoomId,
			event.EventMessage,
			content)
		// These three errors mean we have to make a new Megolm session
		if err == crypto.SessionExpired || err == crypto.SessionNotShared || err == crypto.NoGroupSession {
			log.Debug().Msg("Creating new Megolm session")
			err = state.OlmMachine.ShareGroupSession(
				state.MatrixContext,
				matrixRoomId,
				getUserIDs(state.MatrixContext, state.MautrixClient, matrixRoomId))
			if err != nil {
				log.Fatal().Err(err).Msg("Could not share group session.")
				return
			}
			encryptedContent, err = state.OlmMachine.EncryptMegolmEvent(
				state.MatrixContext,
				matrixRoomId,
				event.EventMessage,
				content)
			if err != nil {
				log.Fatal().Err(err).Msg("Could not encrypt message even after creating new Megolm session")
				return
			}
		}
		if err != nil {
			log.Fatal().Err(err).Msg("Could not encrypt message.")
			return
		}
		eventType = event.EventEncrypted
		eventContent = encryptedContent
	} else {
		eventType = event.EventMessage
		eventContent = content
	}

	_, err := state.MautrixClient.SendMessageEvent(
		state.MatrixContext,
		matrixRoomId,
		eventType,
		eventContent)

	if err != nil {
		log.Fatal().Err(err).Msg("Could not send message to matrix.")
	}

}
