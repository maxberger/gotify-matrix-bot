package send

import (
	"context"
	"gotify_matrix_bot/config"
	"gotify_matrix_bot/gotify_messages"
	"gotify_matrix_bot/matrix"
	"gotify_matrix_bot/template"
	"log"
	"os"
	"strings"

	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func ErrorCallback(err error) {
	panic(err)
}

func Encrypted() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Println("Encryption active")
	ctx := context.Background()

	cli, err := mautrix.NewClient(
		config.Configuration.Matrix.HomeServerURL,
		id.UserID("@"+config.Configuration.Matrix.Username+":"+config.Configuration.Matrix.MatrixDomain),
		config.Configuration.Matrix.Token)

	if err != nil {
		panic(err)
	}

	// DeviceID is needed for some older clients, e.g. some versions Element
	cli.DeviceID = id.DeviceID(strings.ToUpper(random.String(10)))

	cryptoStore := crypto.NewMemoryStore(nil)

	mach := crypto.NewOlmMachine(cli, &zlog.Logger, cryptoStore, &matrix.FakeStateStore{})
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
	// Start long polling in the background
	go func() {
		err = cli.Sync()
		if err != nil {
			panic(err)
		}
	}()

	gotify_messages.OnNewMessage(func(message string) {

		err := matrix.SendEncrypted(ctx, mach, cli, id.RoomID(config.Configuration.Matrix.RoomID), template.GetFormattedMessageString(message))
		if err != nil {
			log.Fatal("Could not send encrypted message to matrix. ", err)
		}

	})

	select {}
}
