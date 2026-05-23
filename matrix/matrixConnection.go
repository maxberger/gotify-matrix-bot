package matrix

import (
	"context"
	"encoding/json"
	"io"
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
	token string,
	downloadFromHostAllowlist []*regexp.Regexp,
) *MatrixState {
	ctx := context.Background()
	cli, err := mautrix.NewClient(homeServerURL, "", "")
	cli.Log = log.With().Str("component", "matrix").Logger()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create matrix client")
	}

	// Intercept /keys/signatures/upload request to mock successful signature upload
	// since publishing cross-signing keys is blocked under MAS/OIDC homeservers.
	transport := cli.Client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	cli.Client.Transport = &signatureInterceptorRoundTripper{underlying: transport}

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

	syncer.OnEvent(func(ctx context.Context, evt *event.Event) {
		if FilterOwnVerificationEvent(evt, cli.UserID) {
			return
		}
		RewriteInRoomVerificationEventID(evt)
	})

	var login *mautrix.ReqLogin
	if token == "" {
		login = &mautrix.ReqLogin{
			Type: mautrix.AuthTypePassword,
			Identifier: mautrix.UserIdentifier{
				Type: mautrix.IdentifierTypeUser,
				User: username,
			},
			Password: password,
		}
	} else {
		login = &mautrix.ReqLogin{
			Type:  mautrix.AuthTypeToken,
			Token: token,
		}
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

	// Load or bootstrap cross-signing keys.
	// The private cross-signing keys are required for the SAS verification
	// flow to complete the trust-signing step (signing the other user's
	// master key). mautrix-go only holds them in-memory, so we persist the
	// seeds to a local file to survive restarts.
	machine := cryptoHelper.Machine()
	loadOrBootstrapCrossSigningKeys(ctx, machine, password)

	acceptAllCallbacks := &AcceptAllVerificationCallbacks{}
	verificationHelper := verificationhelper.NewVerificationHelper(
		cli,
		cryptoHelper.Machine(),
		verificationhelper.NewInMemoryVerificationStore(),
		acceptAllCallbacks,
		/* supportsQRShow= */ false,
		/* supportsQRScan= */ false,
		/* supportsSAS= */ true,
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

func RewriteInRoomVerificationEventID(evt *event.Event) {
	if relatable, ok := evt.Content.Parsed.(event.Relatable); ok {
		if rel := relatable.GetRelatesTo(); rel != nil && rel.Type == event.RelReference && rel.EventID != "" {
			evt.ID = rel.EventID
		}
	}
}

func FilterOwnVerificationEvent(evt *event.Event, ownUserID id.UserID) bool {
	if evt.Sender == ownUserID {
		if strings.HasPrefix(evt.Type.Type, "m.key.verification.") {
			evt.Type = event.Type{Type: "ignored.verification", Class: event.MessageEventType}
			return true
		}
	}
	return false
}

const crossSigningSeedsFile = "crossSigningSeeds.json"

func loadOrBootstrapCrossSigningKeys(ctx context.Context, machine *crypto.OlmMachine, password string) {
	// Try loading persisted seeds from file first.
	seeds, err := loadCrossSigningSeeds()
	if err == nil {
		if err := machine.ImportCrossSigningKeys(seeds); err != nil {
			log.Error().Err(err).Msg("Failed to import cross-signing keys from file")
		} else {
			log.Info().Msg("Cross-signing keys loaded from local file")
			return
		}
	}

	// No persisted seeds — generate new keys.
	if machine.CrossSigningKeys != nil {
		return
	}

	if password == "" {
		log.Warn().Msg("Cross-signing keys not found and no password available to bootstrap them. SAS verification will fail at the trust-signing step.")
		return
	}

	log.Info().Msg("Cross-signing keys not found, bootstrapping...")

	var keysCache *crypto.CrossSigningKeysCache
	var bootstrapErr error
	var recoveryKey string

	// Custom UIA callback that supports both m.login.dummy and robust m.login.password (with identifier object).
	uiaCallback := func(uiResp *mautrix.RespUserInteractive) interface{} {
		flowsJSON, _ := json.Marshal(uiResp.Flows)
		paramsJSON, _ := json.Marshal(uiResp.Params)
		log.Debug().
			Str("session", uiResp.Session).
			RawJSON("flows", flowsJSON).
			RawJSON("params", paramsJSON).
			Str("errcode", uiResp.ErrCode).
			Str("error", uiResp.Error).
			Msg("UIA challenge received from homeserver")

		if uiResp.HasSingleStageFlow(mautrix.AuthTypeDummy) {
			log.Debug().Msg("Responding to UIA with m.login.dummy")
			return &mautrix.BaseAuthData{
				Type:    mautrix.AuthTypeDummy,
				Session: uiResp.Session,
			}
		}

		localpart := machine.Client.UserID.Localpart()
		fullMXID := machine.Client.UserID.String()
		log.Debug().
			Str("user_localpart", localpart).
			Str("user_mxid", fullMXID).
			Msg("Responding to UIA with m.login.password")

		type UIAIdentifier struct {
			Type string `json:"type"`
			User string `json:"user"`
		}
		type UIAAuthPassword struct {
			mautrix.BaseAuthData
			User       string        `json:"user,omitempty"`
			Identifier UIAIdentifier `json:"identifier"`
			Password   string        `json:"password"`
		}

		return &UIAAuthPassword{
			BaseAuthData: mautrix.BaseAuthData{
				Type:    mautrix.AuthTypePassword,
				Session: uiResp.Session,
			},
			User: localpart,
			Identifier: UIAIdentifier{
				Type: "m.id.user",
				User: fullMXID,
			},
			Password: password,
		}
	}

	recoveryKey, keysCache, bootstrapErr = machine.GenerateAndUploadCrossSigningKeys(ctx, uiaCallback, "")
	if bootstrapErr != nil {
		log.Warn().Err(bootstrapErr).Msg("Failed to fully bootstrap cross-signing keys on the server (UIA or SSSS failure)")
		if keysCache == nil {
			log.Info().Msg("Attempting to generate cross-signing keys locally as fallback...")
			keysCache, err = machine.GenerateCrossSigningKeys()
			if err != nil {
				log.Error().Err(err).Msg("Failed to generate cross-signing keys locally")
				return
			}
		} else {
			log.Info().Msg("Using generated cross-signing keys from failed bootstrap attempt")
		}
	} else {
		log.Info().Str("recovery_key", recoveryKey).Msg("Cross-signing keys bootstrapped successfully on the server. Save the recovery key!")
	}

	// Persist the private seeds so they survive restarts.
	exportedSeeds := crypto.CrossSigningSeeds{
		MasterKey:      keysCache.MasterKey.Seed(),
		SelfSigningKey: keysCache.SelfSigningKey.Seed(),
		UserSigningKey: keysCache.UserSigningKey.Seed(),
	}
	if err := saveCrossSigningSeeds(exportedSeeds); err != nil {
		log.Error().Err(err).Msg("Failed to save cross-signing seeds to file")
	}

	// Import the keys so they are loaded into machine.CrossSigningKeys and machine.crossSigningPubkeys
	if err := machine.ImportCrossSigningKeys(exportedSeeds); err != nil {
		log.Error().Err(err).Msg("Failed to import cross-signing keys")
		return
	}

	if bootstrapErr == nil {
		if err := machine.SignOwnDevice(ctx, machine.OwnIdentity()); err != nil {
			log.Error().Err(err).Msg("Failed to sign own device with self-signing key")
		}
		if err := machine.SignOwnMasterKey(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to sign own master key")
		}
	} else {
		log.Warn().Msg("Skipping own device/master key signature upload because cross-signing key publishing failed.")
	}
}

func saveCrossSigningSeeds(seeds crypto.CrossSigningSeeds) error {
	data, err := json.Marshal(seeds)
	if err != nil {
		return err
	}
	return os.WriteFile(crossSigningSeedsFile, data, 0600)
}

func loadCrossSigningSeeds() (crypto.CrossSigningSeeds, error) {
	var seeds crypto.CrossSigningSeeds
	data, err := os.ReadFile(crossSigningSeedsFile)
	if err != nil {
		return seeds, err
	}
	err = json.Unmarshal(data, &seeds)
	return seeds, err
}

type signatureInterceptorRoundTripper struct {
	underlying http.RoundTripper
}

func (rt *signatureInterceptorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/keys/signatures/upload") {
		log.Info().Msg("Intercepted /keys/signatures/upload request, mocking successful response")
		respBody := `{"failures":{}}`
		resp := &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         req.Proto,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			Body:          io.NopCloser(strings.NewReader(respBody)),
			ContentLength: int64(len(respBody)),
			Header:        make(http.Header),
			Request:       req,
		}
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil
	}
	return rt.underlying.RoundTrip(req)
}
