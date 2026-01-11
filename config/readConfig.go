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
	Token         string `yaml:"token,omitempty"`
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
	fatal, warn := checkValues(Configuration)
	if fatal != "" {
		log.Fatal().Msg(fatal)
	}
	if warn != "" {
		log.Warn().Msg(warn)
	}
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

func checkValues(config *Config) (fatal string, warn string) {

	if config.Gotify.URL == "" {
		return "No gotify url specified.", ""
	}

	if config.Gotify.ApiToken == "" {
		return "No gotify api token specified.", ""
	}

	if config.Matrix.HomeServerURL == "" {
		return "No matrix homeserver url specified.", ""
	}

	if config.Matrix.Token == "" {

		if config.Matrix.Username == "" {
			return "No matrix username specified.", ""
		}

		if config.Matrix.Password == "" {
			return "No matrix password specified.", ""
		}

	} else {
		if config.Matrix.Username != "" || config.Matrix.Password != "" {
			return "Matrix token specified along with username/password. Please only specify one authentication method.", ""
		}
	}

	if config.Matrix.RoomID == "" {
		return "No matrix room id specified.", ""
	}
	if config.Debug {
		return "", "Using deprecated keyword 'debug' in config. Please use logging/level instead"
	}
	return "", ""
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
