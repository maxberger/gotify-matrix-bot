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
}

func GetFormattedMessageString(message []byte) string {

	var m Message

	err := json.Unmarshal(message, &m)
	if err != nil {
		log.Error().Err(err).Msgf("Could not parse message from: %s", string(message))
		return "Could not parse message from: " + string(message)
	}

	templateString, err := os.ReadFile("messageTamplate.md")

	if err != nil {
		log.Fatal().Err(err).Msg("Could not find / read messageTamplate.md!")
	}

	content := strings.ReplaceAll(string(templateString), "[TITLE]", m.Title)

	content = strings.ReplaceAll(content, "[MESSAGE]", m.Message)

	return content
}
