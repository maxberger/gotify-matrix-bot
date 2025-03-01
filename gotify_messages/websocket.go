package gotify_messages

import (
	"gotify_matrix_bot/config"
	"net/url"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

type callbackFunction func(string)

func OnNewMessage(callback callbackFunction) {

	websocketURL, urlError := url.Parse(config.Configuration.Gotify.URL + "/stream?token=" + config.Configuration.Gotify.ApiToken)

	if urlError != nil {
		log.Fatal().Err(urlError).Msgf("Error while trying to parse gotify url: %s",
			config.Configuration.Gotify.URL+"/stream?token=[REDACTED]")
	}

	c, _, err := websocket.DefaultDialer.Dial(websocketURL.String(), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while trying to connect to the gotify server.")
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		log.Info().Msg("Connected to Gotify server.")
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Fatal().Err(err).Msg("The websocket connection to gotify returned an error.")
			}

			callback(string(message))

		}
	}()

}
