package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type GotifyType struct {
	URL      string `yaml:"url"`
	ApiToken string `yaml:"apiToken"`
}

type MatrixType struct {
	HomeServerURL string `yaml:"homeserverURL"`
	MatrixDomain  string `yaml:"matrixDomain"`
	Username      string `yaml:"username"`
	Token         string `yaml:"token"`
	RoomID        string `yaml:"roomID"`
	Encrypted     bool   `yaml:"encrypted"`
}
type Config struct {
	Gotify GotifyType
	Matrix MatrixType
	Debug  bool `yaml:"debug"`
}

var Configuration *Config = nil

func LoadConfig() {
	buf, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not load config.")
	}
	Configuration = parseConfig(buf)
}

func ValidateConfig() {
	checkValues(Configuration)
}

func parseConfig(buf []byte) *Config {

	c := &Config{}
	err := yaml.Unmarshal(buf, c)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse config.")
	}

	return fixConfig(c)
}

func fixConfig(c *Config) *Config {
	fixMatrixDomain(c)
	fixGotifyURL(c)
	return c
}

func fixMatrixDomain(c *Config) {
	if c.Matrix.MatrixDomain == "" {
		c.Matrix.MatrixDomain = strings.ReplaceAll(c.Matrix.HomeServerURL, "https://", "")
	}
}

func fixGotifyURL(c *Config) {
	// As the websocket connection for connecting to gotify is used,
	// the scheme is replaced with the appropriate websocket scheme.
	c.Gotify.URL = strings.ReplaceAll(c.Gotify.URL, "http://", "ws://")
	c.Gotify.URL = strings.ReplaceAll(c.Gotify.URL, "https://", "wss://")
	// set default wss scheme for backward compatibility
	if !strings.HasPrefix(c.Gotify.URL, "ws") {
		c.Gotify.URL = "wss://" + c.Gotify.URL
	}
}

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
