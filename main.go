package main

import (
	"gotify_matrix_bot/bot"
	"log"
)

func main() {
	log.Println("The gotify matrix bot has started now.")
	bot.MainLoop()
}
