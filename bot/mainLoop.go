package bot

import (
	"gotify_matrix_bot/config"
	"gotify_matrix_bot/gotify_messages"
	"gotify_matrix_bot/matrix"
	"gotify_matrix_bot/template"

	"github.com/rs/zerolog/log"
)

func MainLoop() {
	log.Info().Msg("Starting main loop...")

	allowListAsRegexps, err := config.DownloadAllowListAsRegexps(config.Configuration)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse allow list")
	}

	matrixConnection := matrix.Connect(
		config.Configuration.Matrix.HomeServerURL,
		config.Configuration.Matrix.Username,
		config.Configuration.Matrix.MatrixDomain,
		config.Configuration.Matrix.Token,
		config.Configuration.Matrix.DeviceID,
		config.Configuration.Matrix.Encrypted,
		allowListAsRegexps,
	)

	gotify_messages.OnNewMessage(func(rawMessage []byte) {
		markdownMessage := template.GetFormattedMessageString(rawMessage)
		messageWithImagesReplaced := matrix.UploadImages(matrixConnection, markdownMessage)
		matrix.SendMessage(
			matrixConnection,
			config.Configuration.Matrix.RoomID, messageWithImagesReplaced,
		)
	})
	select {}
}
