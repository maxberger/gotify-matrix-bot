package config

import (
	"os"
	"regexp"
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
	DeviceID      string `yaml:"deviceID"`
	Encrypted     bool   `yaml:"encrypted"`
}

type LoggingType struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type DownloaderType struct {
	AllowedHosts []string `yaml:"allowedHosts"`
}
type Config struct {
	Gotify  GotifyType
	Matrix  MatrixType
	Logging LoggingType
	// Deprecated: Use Logging instead
	Debug      bool `yaml:"debug"`
	Downloader DownloaderType
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
	fixLoggingLevel(c)
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

func fixLoggingLevel(c *Config) {
	if c.Debug {
		c.Logging.Level = "debug"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
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

	if config.Debug {
		log.Warn().Msg("Using deprecated keyword 'debug' in config. Please use logging/level instead")
	}
}

func DownloadAllowListAsRegexps(config *Config) ([]*regexp.Regexp, error) {
	filters := make([]*regexp.Regexp, len(config.Downloader.AllowedHosts))
	var err error
	for idx, pattern := range config.Downloader.AllowedHosts {
		filters[idx], err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}
	return filters, nil
}
