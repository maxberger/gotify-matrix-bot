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

	matrixConnection := matrix.Connect(
		config.Configuration.Matrix.HomeServerURL,
		config.Configuration.Matrix.Username,
		config.Configuration.Matrix.MatrixDomain,
		config.Configuration.Matrix.Token,
		config.Configuration.Matrix.Encrypted,
	)

	gotify_messages.OnNewMessage(func(message []byte) {
		matrix.SendMessage(
			matrixConnection,
			config.Configuration.Matrix.RoomID,
			template.GetFormattedMessageString(message),
		)
	})
	select {}
}
