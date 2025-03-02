package template

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

type Message struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Extras  struct {
		ClientDisplay struct {
			ContentType string `json:"contentType"`
		} `json:"client::display"`
	} `json:"extras"`
}

func GetFormattedMessageString(message []byte) string {

	var m Message

	err := json.Unmarshal(message, &m)
	if err != nil {
		log.Error().Err(err).Msgf("Could not parse message from: %s", string(message))
		return "Could not parse message from: " + string(message)
	}

	contentType := strings.ToLower(strings.TrimSpace(m.Extras.ClientDisplay.ContentType))

	var markDownContent string

	if len(contentType) == 0 || contentType == "text/plain" {

		templateString, err := os.ReadFile("messageTamplate.md")

		if err != nil {
			log.Fatal().Err(err).Msg("Could not find / read messageTamplate.md!")
			return m.Message
		}

		markDownContent = strings.ReplaceAll(string(templateString), "[TITLE]", m.Title)
		markDownContent = strings.ReplaceAll(markDownContent, "[MESSAGE]", m.Message)
	} else if contentType == "text/markdown" {
		markDownContent = "# " + m.Title + "\n\n" + m.Message
	} else {
		log.Warn().Msgf("Unknown Content Type: %s", contentType)
		markDownContent = m.Message
	}
	return markDownContent
}
