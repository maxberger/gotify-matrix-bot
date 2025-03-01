package matrix

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"

	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type state struct {
	isEncrypted   bool
	matrixContext context.Context
	matrixClient  *mautrix.Client
	olmMachine    *crypto.OlmMachine
}

func Connect(
	homeServerURL string,
	username string,
	domain string,
	token string,
	encrypted bool,
) *state {
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

	return &state{
		isEncrypted:   encrypted,
		matrixContext: ctx,
		matrixClient:  cli,
		olmMachine:    mach,
	}
}

func SendMessage(state *state, roomID string, message string) {
	var err error
	if state.isEncrypted {
		err = SendEncrypted(
			state.matrixContext,
			state.olmMachine,
			state.matrixClient,
			id.RoomID(roomID),
			message)
	} else {
		err = SendUnencrypted(
			state.matrixContext,
			state.matrixClient,
			id.RoomID(roomID),
			message)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Could not send message to matrix.")
	}

}
