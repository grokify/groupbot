package main

import (
	"log"
	"os"

	cfg "github.com/grokify/mogo/config"

	"github.com/grokify/groupbot"

	"github.com/grokify/groupbot/handlers/help"
	"github.com/grokify/groupbot/handlers/help2"
	"github.com/grokify/groupbot/handlers/info"
	"github.com/grokify/groupbot/handlers/list"
	"github.com/grokify/groupbot/handlers/me"
	"github.com/grokify/groupbot/handlers/set"
	"github.com/grokify/groupbot/handlers/stats"
	"github.com/grokify/groupbot/handlers/thanks"
)

func main() {
	// Check and load environment file if necessary
	engine := os.Getenv("GROUPBOT_ENGINE")
	if len(engine) == 0 {
		err := cfg.LoadDotEnvSkipEmpty(os.Getenv("ENV_PATH"), "./.env")
		if err != nil {
			log.Println(err.Error())
		}
		engine = os.Getenv("GROUPBOT_ENGINE")
	}

	// Set intents
	intentRouter := groupbot.IntentRouter{
		Intents: []groupbot.Intent{
			help.NewIntent(),
			help2.NewIntent(),
			info.NewIntent(),
			list.NewIntent(),
			me.NewIntent(),
			stats.NewIntent(),
			thanks.NewIntent(),
			set.NewIntent()}} // Default

	// Run engine
	switch engine {
	case "aws":
		log.Println("Starting Engine [aws-lambda]")
		groupbot.ServeAwsLambda(intentRouter)
	case "nethttp":
		log.Println("Starting Engine [net/http]")
		groupbot.ServeNetHTTP(intentRouter)
	default:
		log.Printf("E_NO_HTTP_ENGINE: [%v]", engine)
	}
}
