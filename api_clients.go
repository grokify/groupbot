package groupbot

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	ru "github.com/grokify/go-ringcentral-client/office/v1/util"
	"github.com/grokify/goauth/authutil"
	gu "github.com/grokify/goauth/google"
	"github.com/grokify/gogoogle/sheetsutil/v4/sheetsmap"
	sheets "google.golang.org/api/sheets/v4"
)

const (
	CharQuoteLeft  = "“"
	CharQuoteRight = "”"
)

type AppConfig struct {
	Port                               int64  `env:"GROUPBOT_PORT"`
	GroupbotRequestFuzzyAtMentionMatch bool   `env:"GROUPBOT_REQUEST_FUZZY_AT_MENTION_MATCH"`
	GroupbotResponseAutoAtMention      bool   `env:"GROUPBOT_RESPONSE_AUTO_AT_MENTION"`
	GroupbotPostSuffix                 string `env:"GROUPBOT_POST_SUFFIX"`
	GroupbotCharQuoteLeft              string `env:"GROUPBOT_CHAR_QUOTE_LEFT"`
	GroupbotCharQuoteRight             string `env:"GROUPBOT_CHAR_QUOTE_RIGHT"`
	RingCentralTokenJSON               string `env:"RINGCENTRAL_TOKEN_JSON"`
	RingCentralServerURL               string `env:"RINGCENTRAL_SERVER_URL"`
	RingCentralWebhookURL              string `env:"RINGCENTRAL_WEBHOOK_URL"`
	RingCentralBotID                   string `env:"RINGCENTRAL_BOT_ID"`
	RingCentralBotName                 string `env:"RINGCENTRAL_BOT_NAME"`
	GoogleSvcAccountJWT                string `env:"GOOGLE_SERVICE_ACCOUNT_JWT"`
	GoogleSpreadsheetID                string `env:"GOOGLE_SPREADSHEET_ID"`
	GoogleSheetTitleRecords            string `env:"GOOGLE_SHEET_TITLE_RECORDS"`
	GoogleSheetTitleMetadata           string `env:"GOOGLE_SHEET_TITLE_METADATA"`
}

func (ac *AppConfig) AppendPostSuffix(s string) string {
	suffix := strings.TrimSpace(ac.GroupbotPostSuffix)
	if len(suffix) > 0 {
		return s + " " + suffix
	}
	return s
}

func (ac *AppConfig) Quote(s string) string {
	return ac.GroupbotCharQuoteLeft + strings.TrimSpace(s) + ac.GroupbotCharQuoteRight
}

func GetRingCentralAPIClient(appConfig AppConfig) (*rc.APIClient, error) {
	fmt.Println(appConfig.RingCentralTokenJSON)
	rcHTTPClient, err := authutil.NewClientTokenJSON(
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
		rcHTTPClient, appConfig.RingCentralServerURL,
	)
}

func GetGoogleAPIClient(appConfig AppConfig) (*http.Client, error) {
	jwtString := appConfig.GoogleSvcAccountJWT
	if len(jwtString) < 1 {
		return nil, fmt.Errorf("no jwt")
	}

	return gu.NewClientFromJWTJSON(
		context.TODO(),
		[]byte(jwtString),
		sheets.DriveScope,
		sheets.SpreadsheetsScope)
}

func GetSheetsMap(googClient *http.Client, spreadsheetID string, sheetTitle string) (sheetsmap.SheetsMap, error) {
	sm, err := sheetsmap.NewSheetsMapTitle(googClient, spreadsheetID, sheetTitle)
	if err != nil {
		return sm, err
	}
	err = sm.FullRead()
	return sm, err
}
