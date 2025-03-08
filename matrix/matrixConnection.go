package matrix

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
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
	token string,
	deviceID string,
	encrypted bool,
	downloadFromHostAllowlist []*regexp.Regexp,
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
	if len(deviceID) == 0 {
		deviceID = strings.ToUpper(random.String(10))
	}
	cli.DeviceID = id.DeviceID(deviceID)

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
		IsEncrypted:               encrypted,
		MatrixContext:             ctx,
		MautrixClient:             cli,
		OlmMachine:                mach,
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
