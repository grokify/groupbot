package groupbot

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	"github.com/grokify/googleutil/sheetsutil/sheetsmap"
	om "github.com/grokify/oauth2more"
	gu "github.com/grokify/oauth2more/google"
	"google.golang.org/api/sheets/v4"
)

type AppConfig struct {
	Port                  int64  `env:"GROUPBOT_PORT"`
	GroupbotName          string `env:"GROUPBOT_NAME"`
	GroupbotAutoAtMention bool   `env:"GROUPBOT_AUTO_AT_MENTION"`
	GroupbotPostSuffix    string `env:"GROUPBOT_POST_SUFFIX"`
	RingCentralTokenJSON  string `env:"RINGCENTRAL_TOKEN_JSON"`
	RingCentralServerURL  string `env:"RINGCENTRAL_SERVER_URL"`
	RingCentralWebhookURL string `env:"RINGCENTRAL_WEBHOOK_URL"`
	RingCentralBotId      string `env:"RINGCENTRAL_BOT_ID"`
	GoogleSvcAccountJWT   string `env:"GOOGLE_SERVICE_ACCOUNT_JWT"`
	GoogleSpreadsheetId   string `env:"GOOGLE_SPREADSHEET_ID"`
	GoogleSheetIndex      int64  `env:"GOOGLE_SHEET_INDEX"`
}

func (ac *AppConfig) AppendPostSuffix(s string) string {
	suffix := strings.TrimSpace(ac.GroupbotPostSuffix)
	if len(suffix) > 0 {
		return s + " " + suffix
	}
	return s
}

func GetRingCentralApiClient(appConfig AppConfig) (*rc.APIClient, error) {
	fmt.Println(appConfig.RingCentralTokenJSON)
	rcHttpClient, err := om.NewClientTokenJSON(
		context.Background(),
		[]byte(appConfig.RingCentralTokenJSON))
	if err != nil {
		return nil, err
	}
	/*
		url := "https://platform.ringcentral.com/restapi/v1.0/glip/groups"
		url = "https://platform.ringcentral.com/restapi/v1.0/subscription"

		resp, err := rcHttpClient.Get(url)
		if err != nil {
			log.Fatal(err)
		} else if resp.StatusCode >= 300 {
			log.Fatal(fmt.Errorf("API Error %v", resp.StatusCode))
		}
	*/
	return ru.NewApiClientHttpClientBaseURL(
		rcHttpClient, appConfig.RingCentralServerURL,
	)
}

func GetGoogleApiClient(appConfig AppConfig) (*http.Client, error) {
	jwtString := appConfig.GoogleSvcAccountJWT
	if len(jwtString) < 1 {
		return nil, fmt.Errorf("No JWT")
	}

	return gu.NewClientFromJWTJSON(
		context.TODO(),
		[]byte(jwtString),
		sheets.DriveScope,
		sheets.SpreadsheetsScope)
}

func GetSheetsMap(googClient *http.Client, appConfig AppConfig) (sheetsmap.SheetsMap, error) {
	sm, err := sheetsmap.NewSheetsMap(
		googClient,
		appConfig.GoogleSpreadsheetId,
		uint(appConfig.GoogleSheetIndex),
	)
	if err != nil {
		return sm, err
	}
	err = sm.FullRead()
	return sm, err
}
