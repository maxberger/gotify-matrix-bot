package config

import (
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type GotifyType struct {
	URL      string `yaml:"url,omitempty"`
	ApiToken string `yaml:"apiToken,omitempty"`
}

type MatrixType struct {
	HomeServerURL string `yaml:"homeserverURL,omitempty"`
	MatrixDomain  string `yaml:"matrixDomain,omitempty"`
	Username      string `yaml:"username,omitempty"`
	Password      string `yaml:"password,omitempty"`
	RoomID        string `yaml:"roomID,omitempty"`
}

type LoggingType struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`
}

type DownloaderType struct {
	AllowedHosts []string `yaml:"allowedHosts,omitempty"`
}
type Config struct {
	Gotify  GotifyType  `yaml:"gotify,omitempty"`
	Matrix  MatrixType  `yaml:"matrix,omitempty"`
	Logging LoggingType `yaml:"logging,omitempty"`
	// Deprecated: Use Logging instead
	Debug      bool           `yaml:"debug,omitempty"`
	Downloader DownloaderType `yaml:"downloader,omitempty"`
}

var Configuration *Config = nil

func InitConfig() {

	buf, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not load config.")
	}

	Configuration = fixConfig(parseConfig(buf))

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

	return c
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

	if config.Matrix.Password == "" {
		log.Fatal().Msg("No matrix password specified.")
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
