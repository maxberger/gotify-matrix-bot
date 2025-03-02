package main

import (
	"gotify_matrix_bot/bot"
	"gotify_matrix_bot/config"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	config.LoadConfig()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if config.Configuration.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	config.ValidateConfig()

	log.Info().Msg("The gotify matrix bot has started now.")
	log.Debug().Msg("Log level is set to debug")
	bot.MainLoop()
}
