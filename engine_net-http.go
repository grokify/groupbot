package groupbot

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grokify/mogo/log/logutil"
	"github.com/grokify/mogo/net/http/httputilmore"
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
	svr := httputilmore.NewServerTimeouts(fmt.Sprintf(":%v", bot.AppConfig.Port), mux, 3*time.Second)
	log.Fatal(svr.ListenAndServe())
}
