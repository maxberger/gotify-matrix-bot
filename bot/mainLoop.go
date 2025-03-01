package bot

import (
	"gotify_matrix_bot/config"
	"gotify_matrix_bot/gotify_messages"
	"gotify_matrix_bot/matrix"
	"gotify_matrix_bot/template"
	"log"
	"os"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func MainLoop() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Printf("Starting main loop; encryption active: %t", config.Configuration.Matrix.Encrypted)

	matrixConnection := matrix.Connect(
		config.Configuration.Matrix.HomeServerURL,
		config.Configuration.Matrix.Username,
		config.Configuration.Matrix.MatrixDomain,
		config.Configuration.Matrix.Token,
		config.Configuration.Matrix.Encrypted,
	)

	gotify_messages.OnNewMessage(func(message string) {
		matrix.SendMessage(
			matrixConnection,
			config.Configuration.Matrix.RoomID,
			template.GetFormattedMessageString(message),
		)
	})
	select {}
}
