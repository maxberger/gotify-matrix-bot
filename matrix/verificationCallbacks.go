package matrix

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/crypto/verificationhelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type VerificationHelperType interface {
	CancelVerification(ctx context.Context, txnID id.VerificationTransactionID, code event.VerificationCancelCode, reason string) error
	AcceptVerification(ctx context.Context, txnID id.VerificationTransactionID) error
	StartSAS(ctx context.Context, txnID id.VerificationTransactionID) error
	ConfirmSAS(ctx context.Context, txnID id.VerificationTransactionID) error
}

type AcceptAllVerificationCallbacks struct {
	VerificationHelper VerificationHelperType
}

var _ verificationhelper.RequiredCallbacks = (*AcceptAllVerificationCallbacks)(nil)
var _ verificationhelper.ShowSASCallbacks = (*AcceptAllVerificationCallbacks)(nil)

func (aavc *AcceptAllVerificationCallbacks) VerificationRequested(ctx context.Context, txnID id.VerificationTransactionID, from id.UserID, fromDevice id.DeviceID) {
	log.Info().
		Str("txnID", txnID.String()).
		Str("from", from.String()).
		Str("device", fromDevice.String()).
		Msg("Verification requested, accepting...")

	go func() {
		err := aavc.VerificationHelper.AcceptVerification(context.Background(), txnID)
		if err != nil {
			log.Error().Err(err).Str("txnID", txnID.String()).Msg("Failed to accept verification")
		}
	}()
}

func (aavc *AcceptAllVerificationCallbacks) VerificationReady(ctx context.Context, txnID id.VerificationTransactionID, otherDeviceID id.DeviceID, supportsSAS, supportsScanQRCode bool, qrCode *verificationhelper.QRCode) {
	log.Info().
		Str("txnID", txnID.String()).
		Str("otherDeviceID", otherDeviceID.String()).
		Msg("Verification ready")
}

func (aavc *AcceptAllVerificationCallbacks) VerificationCancelled(ctx context.Context, txnID id.VerificationTransactionID, code event.VerificationCancelCode, reason string) {
	log.Info().
		Str("txnID", txnID.String()).
		Str("code", string(code)).
		Str("reason", reason).
		Msg("Verification cancelled")
}

// VerificationDone is called when the verification is done.
func (aavc *AcceptAllVerificationCallbacks) VerificationDone(ctx context.Context, txnID id.VerificationTransactionID, method event.VerificationMethod) {
	log.Info().Str("txnID", txnID.String()).Str("method", string(method)).Msg("Verification done")
}

func (aavc *AcceptAllVerificationCallbacks) ShowSAS(ctx context.Context, txnID id.VerificationTransactionID, emojis []rune, emojiDescriptions []string, decimals []int) {
	var emojiList []string
	for i, r := range emojis {
		emojiList = append(emojiList, fmt.Sprintf("%c (%s)", r, emojiDescriptions[i]))
	}

	log.Info().
		Str("txnID", txnID.String()).
		Msgf("Show SAS Emojis: %s", strings.Join(emojiList, " | "))

	log.Info().Str("txnID", txnID.String()).Msg("Automatically confirming SAS emojis...")
	go func() {
		err := aavc.VerificationHelper.ConfirmSAS(context.Background(), txnID)
		if err != nil {
			log.Error().Err(err).Str("txnID", txnID.String()).Msg("Failed to automatically confirm SAS")
		}
	}()
}
