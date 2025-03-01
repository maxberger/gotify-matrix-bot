package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Gotify struct {
		URL      string `yaml:"url"`
		ApiToken string `yaml:"apiToken"`
	}
	Matrix struct {
		HomeServerURL string `yaml:"homeserverURL"`
		MatrixDomain  string `yaml:"matrixDomain"`
		Username      string `yaml:"username"`
		Token         string `yaml:"token"`
		RoomID        string `yaml:"roomID"`
		Encrypted     bool   `yaml:"encrypted"`
	}
	Debug bool `yaml:"debug"`
}

func readConf() *Config {
	buf, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not load config.")
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse config.")
	}

	if c.Matrix.MatrixDomain == "" {
		c.Matrix.MatrixDomain = strings.ReplaceAll(c.Matrix.HomeServerURL, "https://", "")
	}

	// As the websocket connection for connecting to gotify is used,
	// the scheme is replaced with the appropriate websocket scheme.
	c.Gotify.URL = strings.ReplaceAll(c.Gotify.URL, "http://", "ws://")
	c.Gotify.URL = strings.ReplaceAll(c.Gotify.URL, "https://", "wss://")
	// set default wss scheme for backward compatibility
	if !strings.HasPrefix(c.Gotify.URL, "ws") {
		c.Gotify.URL = "wss://" + c.Gotify.URL
	}

	checkValues(c)

	return c
}

var Configuration = readConf()

func checkValues(config *Config) {

	if config.Gotify.URL == "" {
		log.Fatal().Msg("No gotify url specified.")
	}

	if config.Gotify.ApiToken == "" {
		log.Fatal().Msg("No gotify api token specified.")
	}

	if config.Matrix.HomeServerURL == "" {
		log.Fatal().Msg("No matrix homeserver specified.")
	}

	if config.Matrix.Username == "" {
		log.Fatal().Msg("No matrix username specified.")
	}

	if config.Matrix.Token == "" {
		log.Fatal().Msg("No matrix auth token specified.")
	}

	if config.Matrix.RoomID == "" {
		log.Fatal().Msg("No matrix room id specified.")
	}

}
