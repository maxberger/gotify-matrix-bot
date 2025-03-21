package matrix

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/crypto/verificationhelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"

	_ "github.com/mattn/go-sqlite3"
)

type MautrixClientType interface {
	SendMessageEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, content interface{}, extra ...mautrix.ReqSendEvent) (*mautrix.RespSendEvent, error)
	JoinedMembers(ctx context.Context, roomID id.RoomID) (resp *mautrix.RespJoinedMembers, err error)
	UploadMedia(ctx context.Context, data mautrix.ReqUploadMedia) (*mautrix.RespMediaUpload, error)
	Sync() error
}

type MatrixState struct {
	IsEncrypted               bool
	MatrixContext             context.Context
	MautrixClient             MautrixClientType
	OlmMachine                *crypto.OlmMachine
	DownloadFromHostAllowlist []*regexp.Regexp
}

func Connect(
	homeServerURL string,
	username string,
	domain string,
	password string,
	downloadFromHostAllowlist []*regexp.Regexp,
) *MatrixState {
	ctx := context.Background()
	cli, err := mautrix.NewClient(homeServerURL, "", "")
	cli.Log = log.With().Str("component", "matrix").Logger()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create matrix client")
	}

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == cli.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := cli.JoinRoomByID(ctx, evt.RoomID)
			if err == nil {
				log.Info().
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Joined room after invite")
			} else {
				log.Error().Err(err).
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Failed to join room after invite")
			}
		}
	})

	login := &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password: password,
	}

	cryptoHelper, err := cryptohelper.NewCryptoHelper(cli, []byte("gotify-matrix-client"), "cryptoStore.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create Crypto Helper")
	}
	cryptoHelper.LoginAs = login
	log.Debug().Msg("Logging in...")
	err = cryptoHelper.Init(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Login failed.")
	}
	cli.Crypto = cryptoHelper

	acceptAllCallbacks := &AcceptAllVerificationCallbacks{}
	verificationHelper := verificationhelper.NewVerificationHelper(
		cli,
		cryptoHelper.Machine(),
		verificationhelper.NewInMemoryVerificationStore(),
		acceptAllCallbacks,
		false,
		false,
		true,
	)
	acceptAllCallbacks.VerificationHelper = verificationHelper

	err = verificationHelper.Init(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize verification helper")
	}

	// Start long polling in the background
	syncCtx, cancelSync := context.WithCancel(ctx)
	go func() {
		err = cli.SyncWithContext(syncCtx)
		if err != nil {
			log.Fatal().Err(err).Msg("Error during Sync with Matrix server")
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGABRT,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)
	go func() {
		for range c { // when the process is killed
			log.Info().Msg("Cleaning up...")
			cryptoHelper.Close()
			cancelSync()
			os.Exit(0)
		}
	}()

	log.Info().Msgf("Connected to Matrix.")

	return &MatrixState{
		MatrixContext:             ctx,
		MautrixClient:             cli,
		DownloadFromHostAllowlist: downloadFromHostAllowlist,
	}
}

func SendMessage(state *MatrixState, roomID string, markDownMessage string) {
	matrixRoomId := id.RoomID(roomID)

	log.Debug().Msg("Sending message to matrix...")
	log.Debug().Msgf("Message as Markdown: %s", markDownMessage)
	log.Debug().Msgf("Room ID: %s", matrixRoomId)

	content := format.RenderMarkdown(
		markDownMessage,
		/*allowMarkdown = */ true,
		/*allowHTML = */ false)

	_, err := state.MautrixClient.SendMessageEvent(
		state.MatrixContext,
		matrixRoomId,
		event.EventMessage,
		content)

	if err != nil {
		log.Fatal().Err(err).Msg("Could not send message to matrix.")
	}

}

var imageRegexp *regexp.Regexp = regexp.MustCompile(`\!\[\]\(http.*?\)`)

func UploadImages(state *MatrixState, markDownMessage string) string {
	if len(state.DownloadFromHostAllowlist) == 0 {
		return markDownMessage
	}
	leftMost := 0
	for loc := imageRegexp.FindStringIndex(markDownMessage[leftMost:]); loc != nil && leftMost < len(markDownMessage); loc = imageRegexp.FindStringIndex(markDownMessage[leftMost:]) {
		index := loc[0] + leftMost
		end := loc[1] + leftMost
		leftMost = index + 1
		rawUrl := markDownMessage[index+4 : end-1]

		parsed, err := url.Parse(rawUrl)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to parse url %s", rawUrl)
			continue
		}
		allowed := false
		for _, allow := range state.DownloadFromHostAllowlist {
			if allow.MatchString(parsed.Host) {
				allowed = true
				break
			}
		}

		if !allowed {
			log.Info().Msgf("Host is not on allowlist: %s; not downloading %s", parsed.Host, rawUrl)
			continue
		}

		newUrl := downloadAndUploadImage(rawUrl, state)
		log.Debug().Msgf("Image was stored as %s", newUrl)

		replacement := "![](" + newUrl + ")"
		markDownMessage = markDownMessage[:index] + string(replacement) + markDownMessage[end:]
	}
	return markDownMessage
}

func downloadAndUploadImage(url string, state *MatrixState) string {
	log.Debug().Msgf("Downloading image from %s", url)
	downloadRespose, err := http.Get(url)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to download %s", url)
		return url
	}

	contentType := downloadRespose.Header.Get("Content-Type")
	contentLength := downloadRespose.ContentLength

	log.Debug().Msgf("Found image of type %s with length %d", contentType, contentLength)

	if !strings.HasPrefix(contentType, "image/") {
		log.Warn().Err(err).Msgf("Invalid image content type: %s", contentType)
		return url
	}

	resp, err := state.MautrixClient.UploadMedia(state.MatrixContext, mautrix.ReqUploadMedia{
		Content:       downloadRespose.Body,
		ContentLength: contentLength,
		ContentType:   contentType,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to upload image")
		return url
	}
	newUrl := resp.ContentURI.CUString()
	return string(newUrl)
}
