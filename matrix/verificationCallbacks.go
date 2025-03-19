package matrix

import (
	"context"

	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/crypto/verificationhelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type AcceptAllVerificationCallbacks struct {
	VerificationHelper *verificationhelper.VerificationHelper
}

func (aavc *AcceptAllVerificationCallbacks) VerificationRequested(ctx context.Context, txnID id.VerificationTransactionID, from id.UserID, fromDevice id.DeviceID) {
	log.Info().Str("txnID", txnID.String()).Msg("Verification requested, rejecting...")
	aavc.VerificationHelper.CancelVerification(ctx, txnID, event.VerificationCancelCodeUser, "Verification is not yet supported")
}

func (aavc *AcceptAllVerificationCallbacks) VerificationReady(ctx context.Context, txnID id.VerificationTransactionID, otherDeviceID id.DeviceID, supportsSAS, supportsScanQRCode bool, qrCode *verificationhelper.QRCode) {
	log.Info().Str("txnID", txnID.String()).Msg("Verification ready")

}

func (aavc *AcceptAllVerificationCallbacks) VerificationCancelled(ctx context.Context, txnID id.VerificationTransactionID, code event.VerificationCancelCode, reason string) {
	log.Info().Str("txnID", txnID.String()).Msg("Verification cancelled")
}

// VerificationDone is called when the verification is done.
func (aavc *AcceptAllVerificationCallbacks) VerificationDone(ctx context.Context, txnID id.VerificationTransactionID) {
	log.Info().Str("txnID", txnID.String()).Msg("Verification done")
}

func (aavc *AcceptAllVerificationCallbacks) ShowSAS(ctx context.Context, txnID id.VerificationTransactionID, emojis []rune, emojiDescriptions []string, decimals []int) {
	log.Info().Str("txnID", txnID.String()).Msgf("Show SAS %v %v %v", emojis, emojiDescriptions, decimals)
}
