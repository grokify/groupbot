package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/caarlos0/env"
	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	"github.com/grokify/googleutil/sheetsutil"
	"github.com/grokify/googleutil/sheetsutil/sheetsmap"
	cfg "github.com/grokify/gotilla/config"
	"github.com/grokify/gotilla/encoding/jsonutil"
	"github.com/grokify/gotilla/html/htmlutil"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/groupbot/config"
)

const ValidationTokenHeader = "Validation-Token"

func StripAtMention(id, text string) string {
	rx := regexp.MustCompile(fmt.Sprintf("!\\[:Person\\]\\(%v\\)", id))
	noAtMention := rx.ReplaceAllString(text, " ")
	noAtMention = regexp.MustCompile(`\\s+`).ReplaceAllString(noAtMention, " ")
	return strings.TrimSpace(noAtMention)
}

func HandleWebhookNetHTTP(res http.ResponseWriter, req *http.Request) {
}

type AnyHTTPHandler struct {
	Port              int
	AppConfig         config.AppConfig
	RingCentralClient *rc.APIClient
	GoogleClient      *http.Client
	SheetsMap         sheetsmap.SheetsMap
}

var anyHTTPHandler = AnyHTTPHandler{
	Port: 8080,
}

type EventResponse struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Message    string            `json:"message,omitempty"`
}

func (er *EventResponse) ToJson() []byte {
	if len(er.Message) == 0 {
		er.Message = ""
	}
	msgJson, err := json.Marshal(er)
	if err != nil {
		return []byte(`{"statusCode":500,"message":"Cannot Marshal to JSON"}`)
	}
	return msgJson
}

func (h *AnyHTTPHandler) ItemToGlipCreatePost(postText string, item sheetsmap.Item) rc.GlipCreatePost {
	bodyFields := []rc.GlipMessageAttachmentFieldsInfo{}

	numPrefixColumns := 2
	haveItems := 0
	color := htmlutil.Color2GreenHex

	for i, col := range h.SheetsMap.Columns {
		if i < numPrefixColumns {
			continue
		}

		userValue := ""
		if userValueTry, ok := item.Data[col.Value]; ok {
			userValue = strings.TrimSpace(userValueTry)
		}

		if len(userValue) > 0 {
			haveItems += 1
		}

		bodyFields = append(bodyFields, rc.GlipMessageAttachmentFieldsInfo{
			Title: col.Value,
			Value: userValue,
		})
	}
	if haveItems == 0 {
		color = htmlutil.Color2RedHex
	} else if haveItems < (len(h.SheetsMap.Columns) - numPrefixColumns) {
		color = htmlutil.Color2YellowHex
	}
	return rc.GlipCreatePost{
		Text: postText,
		Attachments: []rc.GlipMessageAttachmentInfoRequest{
			{
				Type_:  "Card",
				Color:  color,
				Fields: bodyFields,
			},
		},
	}
}

func (h *AnyHTTPHandler) ItemValueToGlipCreatePost(postText string, item sheetsmap.Item, colName string) rc.GlipCreatePost {
	bodyFields := []rc.GlipMessageAttachmentFieldsInfo{}

	numPrefixColumns := 2
	haveItems := 0
	color := htmlutil.Color2GreenHex

	for i, col := range h.SheetsMap.Columns {
		if i < numPrefixColumns {
			continue
		}
		if strings.ToLower(col.Value) != strings.ToLower(colName) {
			continue
		}

		userValue := ""
		if userValueTry, ok := item.Data[col.Value]; ok {
			userValue = strings.TrimSpace(userValueTry)
		}

		if len(userValue) > 0 {
			haveItems += 1
		}

		bodyFields = append(bodyFields, rc.GlipMessageAttachmentFieldsInfo{
			Title: col.Value,
			Value: userValue,
		})
	}
	if haveItems == 0 {
		color = htmlutil.Color2RedHex
	}
	return rc.GlipCreatePost{
		Text: postText,
		Attachments: []rc.GlipMessageAttachmentInfoRequest{
			{
				Type_:  "Card",
				Color:  color,
				Fields: bodyFields,
			},
		},
	}
}

func (h *AnyHTTPHandler) StatsToGlipCreatePost() (rc.GlipCreatePost, error) {
	reqBody := rc.GlipCreatePost{}
	stats, err := h.SheetsMap.CombinedStatsCol0Enum()
	if err != nil {
		return reqBody, err
	}

	statsTexts := []string{}
	for _, stat := range stats {
		statsText := fmt.Sprintf("%v - %s", stat.Count, stat.Name)
		statsTexts = append(statsTexts, statsText)
	}
	statsTextsString := ""
	if len(statsTexts) > 0 {
		colKeys := h.SheetsMap.DataColumnsKeys()
		header := "count - " + strings.Join(colKeys, ", ")
		statsTextsString = header + "\n* " + strings.Join(statsTexts, "\n* ")
	}
	reqBody.Text = "Here's the current stats:"
	reqBody.Attachments = []rc.GlipMessageAttachmentInfoRequest{{
		Type_: "Card",
		Color: htmlutil.RingCentralOrangeHex,
		Text:  statsTextsString,
	}}

	return reqBody, nil
}

func (h *AnyHTTPHandler) ListAsGlipCreatePost() rc.GlipCreatePost {
	displays := []string{}
	keys := []string{}
	keysMap := map[string]string{}
	for _, item := range h.SheetsMap.ItemMap {
		displays = append(displays, item.Display)
		keys = append(keys, item.Key)
		vals := []string{}
		for _, col := range h.SheetsMap.DataColumnsKeys() {
			if itemVal, ok := item.Data[col]; ok {
				itemVal = strings.TrimSpace(itemVal)
				if len(itemVal) > 0 {
					vals = append(vals, itemVal)
				} else {
					vals = append(vals, "?")
				}
			} else {
				vals = append(vals, "?")
			}
		}
		itemString := item.Display + " - " + strings.Join(vals, ", ")
		keysMap[item.Key] = itemString
	}
	sort.Strings(displays)

	outputs := []string{}

	for i, _ := range displays {
		if i < len(keys) {
			key := keys[i]
			if output, ok := keysMap[key]; ok {
				outputs = append(outputs, output)
			}
		}
	}

	outputsString := "* " + strings.Join(outputs, "\n* ")

	return rc.GlipCreatePost{
		Text: "Here's the current data:",
		Attachments: []rc.GlipMessageAttachmentInfoRequest{{
			Type_: "Card",
			Color: htmlutil.RingCentralOrangeHex,
			Text:  outputsString,
		}},
	}
}

func (h *AnyHTTPHandler) HelpAsGlipCreatePost(numGroupMembers int64) rc.GlipCreatePost {
	reqBody := rc.GlipCreatePost{}

	colNames := []string{}
	haveEnums := false
	exampleKey := ""
	exampleVal := ""

	for i, col := range h.SheetsMap.Columns {
		if i < 2 {
			continue
		}

		attachment := rc.GlipMessageAttachmentInfoRequest{
			Type_:  "Card",
			Fields: []rc.GlipMessageAttachmentFieldsInfo{},
		}

		attachment.Fields = append(attachment.Fields,
			rc.GlipMessageAttachmentFieldsInfo{
				Title: "Field",
				Value: col.Value,
				Style: "Short",
			})
		if len(exampleKey) == 0 {
			exampleKey = col.Value
		}
		colNames = append(colNames, col.Value)
		if len(col.Enums) > 0 {
			haveEnums = true
			options := []string{}
			for _, enum := range col.Enums {
				parts := []string{}
				if len(enum.Canonical) > 0 {
					parts = append(parts, enum.Canonical)
					if len(exampleVal) == 0 {
						exampleVal = enum.Canonical
					}
				}
				if len(enum.Aliases) > 0 {
					aliasesStr := fmt.Sprintf("(%v)", strings.Join(enum.Aliases, ", "))
					parts = append(parts, aliasesStr)
				}
				if len(parts) > 0 {
					options = append(options, strings.Join(parts, " "))
				}
			}
			if len(options) > 0 {
				optionsStr := ""
				if len(options) == 1 {
					optionsStr = options[1]
				} else {
					optionsStr = "* " + strings.Join(options, "\n* ")
				}
				attachment.Fields = append(attachment.Fields,
					rc.GlipMessageAttachmentFieldsInfo{
						Title: "Values (with aliases)",
						Value: optionsStr,
						Style: "Short",
					})
			}
		}
		if len(attachment.Fields) > 0 {
			mod := math.Mod(float64(len(reqBody.Attachments)), 2)
			if mod == 0 {
				attachment.Color = htmlutil.RingCentralOrangeHex
			} else {
				attachment.Color = htmlutil.RingCentralBlueHex
			}
			reqBody.Attachments = append(reqBody.Attachments, attachment)
		}
	}
	if len(reqBody.Attachments) > 0 {
		enumsText := ""
		if haveEnums {
			enumsText = " See more on values you can use below below."
		}
		exampleText := ""
		if len(exampleKey) > 0 && len(exampleVal) > 0 {
			exampleText = ", for example: `" + exampleKey + " " + exampleVal + "`"
		}

		reqBody.Text = fmt.Sprintf("Hi there, I'm here to help you share some data. Here are some things you can do use me:\n\n* You can set the following fields: %s\n* To set a field, say `<field> <value>`%s.%s\n* Additional commands include `me`, `list`, `stats`, and `info`\n* If there are more than 2 people in our conversation, you will need to @ mention me", strings.Join(colNames, ", "), exampleText, enumsText)
	}
	return reqBody
}

func (h *AnyHTTPHandler) ProcessEvent(reqBodyBytes []byte) (EventResponse, error) {
	evt := &ru.Event{}
	err := json.Unmarshal(reqBodyBytes, evt)
	log.Info(string(reqBodyBytes))
	if err != nil {
		log.Warn(fmt.Sprintf("Request Bytes: %v", string(reqBodyBytes)))
		log.Warn(fmt.Sprintf("Cannot Unmarshal to Event: %s", err.Error()))
		return EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("400 Cannot Unmarshal to Event: %s", err.Error()),
		}, fmt.Errorf("JSON Unmarshal Error: %s", err.Error())
	}

	if !evt.IsEventType(ru.GlipPostEvent) {
		return EventResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	glipPostEvent, err := evt.GetGlipPostEventBody()
	if err != nil {
		log.Warn(err)
		return EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("400 Cannot unmarshal to GlipPostEvent: %v", err.Error()),
		}, nil
	}
	log.Info(string(jsonutil.MustMarshal(glipPostEvent, true)))
	if (glipPostEvent.EventType != "PostAdded" &&
		glipPostEvent.EventType != "PostChanged") ||
		glipPostEvent.Type_ != "TextMessage" ||
		glipPostEvent.CreatorId == h.AppConfig.RingCentralBotId {

		log.Info("E_NOT_PostAdded or TextMessage")
		return EventResponse{
			StatusCode: http.StatusOK,
			Message:    "200 Not a relevant post: Not PostAdded|PostChanged && TextMessage",
		}, nil
	}

	glipApiUtil := ru.GlipApiUtil{ApiClient: h.RingCentralClient}
	groupMemberCount, _ := glipApiUtil.GlipGroupMemberCount(glipPostEvent.GroupId)
	log.Info(fmt.Sprintf("GROUP_MEMBER_COUNT [%v]", groupMemberCount))
	atMentionedOrGroupOfTwo, err := glipApiUtil.AtMentionedOrGroupOfTwo(
		h.AppConfig.RingCentralBotId,
		glipPostEvent.GroupId,
		glipPostEvent.Mentions)
	if err != nil {
		return EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    "500 AtMentionedOrGroupOfTwo error",
		}, nil
	}
	if !atMentionedOrGroupOfTwo {
		log.Info("E_NO_MENTION")
		return EventResponse{
			StatusCode: http.StatusOK,
			Message:    "200 Not Mentioned in a Group != 2 members",
		}, nil
	}

	creator, resp, err := h.RingCentralClient.GlipApi.LoadPerson(
		context.Background(), glipPostEvent.CreatorId)
	if err != nil {
		msg := fmt.Errorf("Glip API Load Person Error: %v", err.Error())
		log.Warn(msg.Error())
		return EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    msg.Error(),
		}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("Glip API Status Error: %v", resp.StatusCode)
		log.Warn(msg.Error())
		return EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}

	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Info(fmt.Sprintf("Poster [%v][%v]", name, email))

	text := strings.TrimSpace(StripAtMention(
		h.AppConfig.RingCentralBotId, glipPostEvent.Text))
	textLc := strings.ToLower(text)

	reqBody := rc.GlipCreatePost{}

	// Process Intents
	if textLc == "help" {
		log.Info("INTENT [Help]")
		reqBody = h.HelpAsGlipCreatePost(int64(groupMemberCount))
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	} else if textLc == "info" {
		log.Info("INTENT [Info]")
		spreadsheetURL := sheetsutil.SheetToWebURL(h.AppConfig.GoogleSpreadsheetId)
		reqBody = rc.GlipCreatePost{
			Text: fmt.Sprintf("I am a bot accessing this Google sheet:\n\n%s\n\nI was created by [grokify](https://github.com/grokify).", spreadsheetURL),
		}
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	} else if textLc == "me" {
		log.Info("INTENT [Me]")
		item, err := h.SheetsMap.GetItem(email)
		if err != nil {
			msg := fmt.Errorf("Cannot get item from sheet: [%v]", email)
			log.Warn(msg.Error())
			return EventResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 " + msg.Error(),
			}, err
		}
		if item.Display != name {
			item.Display = name
			h.SheetsMap.SynchronizeItem(item)
		}
		reqBody = h.ItemToGlipCreatePost(fmt.Sprintf("Here's your info, %v", name), item)
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	} else if textLc == "list" {
		reqBody := h.ListAsGlipCreatePost()
		if err != nil {
			msg := fmt.Errorf("Cannot get stats from sheet: [%v]", err.Error())
			log.Warn(msg.Error())
			return EventResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 " + msg.Error(),
			}, err
		}
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	} else if textLc == "stats" {
		reqBody, err = h.StatsToGlipCreatePost()
		if err != nil {
			msg := fmt.Errorf("Cannot get stats from sheet: [%v]", err.Error())
			log.Warn(msg.Error())
			return EventResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 " + msg.Error(),
			}, err
		}
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	} else {
		for _, col := range h.SheetsMap.Columns {
			if textLc == strings.ToLower(col.Value) {
				item, err := h.SheetsMap.GetItem(email)
				if err != nil {
					msg := fmt.Errorf("Cannot get item from sheet: [%v]", email)
					log.Warn(msg.Error())
					return EventResponse{
						StatusCode: http.StatusInternalServerError,
						Message:    "500 " + msg.Error(),
					}, err
				}
				if item.Display != name {
					item.Display = name
					h.SheetsMap.SynchronizeItem(item)
				}
				reqBody = h.ItemValueToGlipCreatePost(fmt.Sprintf("Here's your info, %v. Use `me` to see all your data.", name), item, col.Value)
				return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
			}
		}

		log.Info(fmt.Sprintf("INTENT [Set] EMAIL[%v] TEXT[%v]", email, text))
		err := h.SheetsMap.SetItemKeyDisplay(email, name)
		if err != nil {
			msg := fmt.Errorf("E_CANNOT_SET_NAME: KEY[%v] NAME[%v] ERR[%v]", email, name, err.Error())
			log.Warn(msg.Error())
			return EventResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 " + msg.Error(),
			}, err
		}
		_, err = h.SheetsMap.SetItemKeyString(email, text)
		if err != nil {
			msg := fmt.Errorf("E_CANNOT_ADD_TO_SHEET: KEY[%v] VAL[%v]", email, text)
			log.Warn(msg.Error())
			namePrefix := ""
			if len(name) > 0 {
				namePrefix = name + ", "
			}
			reqBody = rc.GlipCreatePost{
				Text: fmt.Sprintf("%sI couldn't understand you. Please type `help` to get more information on how I can help. Remember to @ mention me if our conversation has more than the two of us.", namePrefix),
			}
			return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
		}

		item, err := h.SheetsMap.GetItem(email)
		if err != nil {
			msg := fmt.Errorf("Cannot get item from sheet: [%v]", email)
			log.Warn(msg.Error())
			return EventResponse{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 " + msg.Error(),
			}, err
		}

		reqBody = h.ItemToGlipCreatePost(fmt.Sprintf("Thanks for the update %v. Here's your info", name), item)
		return h.SendGlipPost(glipPostEvent.GroupId, reqBody)
	}
}

func (h *AnyHTTPHandler) SendGlipPost(groupId string, reqBody rc.GlipCreatePost) (EventResponse, error) {
	_, resp, err := h.RingCentralClient.GlipApi.CreatePost(
		context.Background(), groupId, reqBody,
	)
	if err != nil {
		msg := fmt.Errorf("Cannot Create Post: [%v]", err.Error())
		log.Warn(msg.Error())
		return EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("Cannot Create Post, API Status [%v]", resp.StatusCode)
		log.Warn(msg.Error())
		return EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}
	return EventResponse{}, nil
}

func (h *AnyHTTPHandler) HandleAwsLambda(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info("Handling Lambda Request")
	log.Info(fmt.Sprintf("REQ_BODY: %v", req.Body))
	/*
		vt := req.Header.Get(ValidationTokenHeader)
		if len(strings.TrimSpace(vt)) > 0 {
			res.Header().Set(ValidationTokenHeader, vt)
			return
		}
	*/
	/*
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{},
			Body:       `{"statusCode":200,"body":"Testing."}`,
		}, nil
	*/
	evtResp, err := h.Initialize()
	if err != nil {
		body := `{"statusCode":500,"body":"Cannot initialize."}`
		log.Info(body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{},
			Body:       `{"statusCode":500,"body":"Cannot initialize."}`,
		}, nil
	}

	if vt, ok := req.Headers[ValidationTokenHeader]; ok {
		body := `{"statusCode":200}`
		log.Info(body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{ValidationTokenHeader: vt},
			Body:       `{"statusCode":200}`,
		}, nil
	}
	evtResp, _ = h.ProcessEvent([]byte(req.Body))

	awsRespBody := strings.TrimSpace(string(evtResp.ToJson()))
	log.Info("RESP_BODY: %v", awsRespBody)
	if len(awsRespBody) == 0 ||
		strings.Index(awsRespBody, "{") != 0 {
		awsRespBody = `{"statusCode":500}`
	}

	awsResp := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{},
		Body:       awsRespBody}
	return awsResp, nil
}

func (h *AnyHTTPHandler) HandleNetHTTP(res http.ResponseWriter, req *http.Request) {
	// Check for RingCentral Validation-Token setup
	vt := req.Header.Get(ValidationTokenHeader)
	if len(strings.TrimSpace(vt)) > 0 {
		res.Header().Set(ValidationTokenHeader, vt)
		return
	}
	_, err := h.Initialize()
	if err != nil {
		log.Warn(err)
	}

	reqBodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warn(err)
	}

	evtResp, err := h.ProcessEvent(reqBodyBytes)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	} else {
		res.WriteHeader(evtResp.StatusCode)
	}
}

func serveNetHttp() {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", http.HandlerFunc(anyHTTPHandler.HandleNetHTTP))
	mux.HandleFunc("/webhook/", http.HandlerFunc(anyHTTPHandler.HandleNetHTTP))
	fmt.Println("FINISH_MUX")
	fmt.Println(anyHTTPHandler.Port)
	/*
		done := make(chan bool)
		go http.ListenAndServe(":"+port, nil)
		log.Printf("Server started at port %v", port)
		<-done
	*/
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", anyHTTPHandler.Port), mux))
}

func (h *AnyHTTPHandler) Initialize() (EventResponse, error) {
	appCfg := config.AppConfig{}
	err := env.Parse(&appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Cannot Parse Config: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Cannot Parse Config: %v", err.Error()),
		}, err
	}
	h.AppConfig = appCfg

	log.Info(fmt.Sprintf("BOT_ID: %v", h.AppConfig.RingCentralBotId))

	rcApiClient, err := config.GetRingCentralApiClient(appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: RC Client: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: RC Client: %v", err.Error()),
		}, err
	}
	h.RingCentralClient = rcApiClient

	googHttpClient, err := config.GetGoogleApiClient(appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Google Client: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Google Client: %v", err.Error()),
		}, err
	}
	h.GoogleClient = googHttpClient

	sm, err := config.GetSheetsMap(googHttpClient, appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()),
		}, err
	}
	h.SheetsMap = sm

	return EventResponse{
		StatusCode: 200,
		Message:    "Initialize success",
	}, nil
}

func HandleAwsLambda(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info("HandleAwsLambda_S1")
	log.Info(fmt.Sprintf("HandleAwsLambda_S2_req_body: %v", req.Body))

	if val, ok := req.Headers[ValidationTokenHeader]; ok {
		if len(val) > 0 {
			return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Headers:    map[string]string{ValidationTokenHeader: val},
				Body:       `{"statusCode":200}`,
			}, nil
		}
	}

	anyHTTPHandler := AnyHTTPHandler{}
	return anyHTTPHandler.HandleAwsLambda(req)
}

func serveAwsLambda() {
	log.Info("serveAwsLambda_S1")
	lambda.Start(HandleAwsLambda)
}

func main() {
	if 1 == 0 {
		awsResp := events.APIGatewayProxyResponse{
			StatusCode: 204,
			Headers:    map[string]string{},
			Body:       `{"statusCode":204}`}
		str, err := json.Marshal(awsResp)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(str))
		panic(string("Z"))
	}

	log.Info(os.Getenv("GOOGLE_SPREADSHEET_ID"))

	engine := os.Getenv("DATABOT_ENGINE")

	if len(engine) == 0 {
		err := cfg.LoadDotEnvSkipEmpty(os.Getenv("ENV_PATH"), "./.env")
		if err != nil {
			log.Info(err)
		}
		engine = os.Getenv("DATABOT_ENGINE")
	}

	switch engine {
	case "aws":
		log.Info("Starting Lambda engine")
		serveAwsLambda()
	case "nethttp":
		serveNetHttp()
	default:
		log.Info(fmt.Sprintf("E_NO_HTTP_ENGINE: [%v]", engine))
	}
}
