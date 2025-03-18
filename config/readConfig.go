package config

import (
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mau.fi/util/random"

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
	Token         string `yaml:"token,omitempty"`
	RoomID        string `yaml:"roomID,omitempty"`
	DeviceID      string `yaml:"deviceID,omitempty"`
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
	bufG, err := os.ReadFile("./config.generated.yaml")
	var genConfig *Config = nil
	if err == nil {
		genConfig = parseConfig(bufG)
	}

	buf, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not load config.")
	}
	var fixes *Config = nil

	Configuration, fixes = fixConfig(parseConfig(buf), genConfig)

	storeConfigFixes(fixes)
}

func storeConfigFixes(fixes *Config) {
	out, err := yaml.Marshal(fixes)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not generate config.")
	}
	os.WriteFile("./config.generated.yaml", out, 0644)
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

func fixConfig(c *Config, generatedConfig *Config) (*Config, *Config) {
	fixesToStore := &Config{}
	fixMatrixDomain(c)
	fixGotifyURL(c)
	fixLoggingLevel(c)
	fixDeviceId(c, generatedConfig, fixesToStore)
	return c, fixesToStore
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

func fixDeviceId(c *Config, generatedConfig *Config, fixesToStore *Config) {
	if len(c.Matrix.DeviceID) == 0 {
		var deviceID string
		if len(generatedConfig.Matrix.DeviceID) > 0 {
			deviceID = generatedConfig.Matrix.DeviceID
		} else {
			deviceID = strings.ToUpper(random.String(10))
		}
		fixesToStore.Matrix.DeviceID = deviceID
		c.Matrix.DeviceID = deviceID
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
