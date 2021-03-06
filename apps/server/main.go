package main

import (
	"fmt"
	"os"

	cfg "github.com/grokify/gotilla/config"
	log "github.com/sirupsen/logrus"

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
			log.Warn(err)
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
		log.Info("Starting Engine [aws-lambda]")
		groupbot.ServeAwsLambda(intentRouter)
	case "nethttp":
		log.Info("Starting Engine [net/http]")
		groupbot.ServeNetHttp(intentRouter)
	default:
		log.Warn(fmt.Sprintf("E_NO_HTTP_ENGINE: [%v]", engine))
	}
}
