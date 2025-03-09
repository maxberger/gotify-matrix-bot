package main

import (
	"gotify_matrix_bot/bot"
	"gotify_matrix_bot/config"
	"os"
	"runtime/debug"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	config.LoadConfig()
	setupLoggerFromConfig()
	config.ValidateConfig()

	logVersion()
	bot.MainLoop()
}

func logVersion() {
	info, ok := debug.ReadBuildInfo()

	versionString := ""
	if ok {
		versionString = " " + info.Main.Version
	}

	log.Info().Msgf("Gotify matrix bot%s has started.", versionString)
}

func setupLoggerFromConfig() {
	switch {
	case strings.EqualFold(config.Configuration.Logging.Format, "json"):
		// Nothing to do, this is the default in zerolog
	case strings.EqualFold(config.Configuration.Logging.Format, "plain") || config.Configuration.Logging.Format == "":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
	case strings.EqualFold(config.Configuration.Logging.Format, "color"):
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false})
	default:
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
		log.Warn().Msgf("Unknown log format %s, defaulting to plain", config.Configuration.Logging.Format)
	}

	level, err := zerolog.ParseLevel(config.Configuration.Logging.Level)

	if err != nil {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unknown log level %s, defaulting to info", config.Configuration.Logging.Level)
	} else {
		zerolog.SetGlobalLevel(level)
		log.Debug().Msgf("Log level set to %s", level)
	}

}
