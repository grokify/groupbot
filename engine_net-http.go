package groupbot

import (
	"fmt"
	"log"
	"net/http"

	"github.com/grokify/mogo/log/logutil"
)

func ServeNetHTTP(intentRouter IntentRouter) {
	bot := Groupbot{}
	_, err := bot.Initialize()
	logutil.FatalErr(err)
	bot.IntentRouter = intentRouter

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", http.HandlerFunc(bot.HandleNetHTTP))
	mux.HandleFunc("/webhook/", http.HandlerFunc(bot.HandleNetHTTP))

	log.Printf("Starting server on port [%v]", bot.AppConfig.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", bot.AppConfig.Port), mux))
}
