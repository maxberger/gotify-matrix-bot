package template

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
)

type Message struct {
	Id      int64  `json:"id"`
	AppId   int64  `json:"appid"`
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
	log.Info().Msgf("Forwarding message %d/%d with Title %s", m.AppId, m.Id, m.Title)

	contentType := strings.ToLower(strings.TrimSpace(m.Extras.ClientDisplay.ContentType))

	var markDownContent string

	if len(contentType) == 0 || contentType == "text/plain" {
		templateString := `# [TITLE]

[MESSAGE]
`

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
