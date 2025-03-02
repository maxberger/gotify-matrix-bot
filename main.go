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
	setupLoggerFromConfig()
	config.ValidateConfig()

	log.Info().Msg("The gotify matrix bot has started now.")
	bot.MainLoop()
}

func setupLoggerFromConfig() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	level, err := zerolog.ParseLevel(config.Configuration.Logging.Level)

	if err != nil {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unknown log level %s, defaulting to info", config.Configuration.Logging.Level)
	} else {
		zerolog.SetGlobalLevel(level)
		log.Debug().Msgf("Log level set to %s", level)
	}

}
