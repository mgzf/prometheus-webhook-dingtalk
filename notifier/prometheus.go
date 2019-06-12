package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/timonwong/prometheus-webhook-dingtalk/models"
	"github.com/timonwong/prometheus-webhook-dingtalk/template"
)

var (
	keywordLeader = map[string]string{}
)

func LoadMobileFromFile(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("ReadFile: ", err.Error())
		return err
	}

	if err := json.Unmarshal(bytes, &keywordLeader); err != nil {
		fmt.Println("Unmarshal: ", err.Error())
		return err
	}

	return nil
}

func BuildDingTalkNotification(promMessage *models.WebhookMessage) (*models.DingTalkNotification, error) {
	title, err := template.ExecuteTextString(`{{ template "ding.link.title" . }}`, promMessage)
	if err != nil {
		return nil, err
	}
	content, err := template.ExecuteTextString(`{{ template "ding.link.content" . }}`, promMessage)
	if err != nil {
		return nil, err
	}
	var buttons []models.DingTalkNotificationButton
	for i, alert := range promMessage.Alerts.Firing() {
		buttons = append(buttons, models.DingTalkNotificationButton{
			Title:     fmt.Sprintf("Graph for alert #%d", i+1),
			ActionURL: alert.GeneratorURL,
		})
	}

	var atMobiles []string
	for keyword := range keywordLeader {
		if strings.Contains(content, keyword) {
			mobile := keywordLeader[keyword]
			atMobiles = append(atMobiles, mobile)
			content += " @" + mobile + " "
		}
	}

	notification := &models.DingTalkNotification{
		MessageType: "markdown",
		Markdown: &models.DingTalkNotificationMarkdown{
			Title: title,
			Text:  content,
		},
		At: &models.DingTalkNotificationAt{
			AtMobiles: atMobiles,
		},
	}

	return notification, nil
}

func SendDingTalkNotification(httpClient *http.Client, webhookURL string, notification *models.DingTalkNotification) (*models.DingTalkNotificationResponse, error) {
	body, err := json.Marshal(&notification)
	if err != nil {
		return nil, errors.Wrap(err, "error encoding DingTalk request")
	}

	httpReq, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "error building DingTalk request")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	req, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending notification to DingTalk")
	}
	defer req.Body.Close()

	if req.StatusCode != 200 {
		return nil, errors.Errorf("unacceptable response code %d", req.StatusCode)
	}

	var robotResp models.DingTalkNotificationResponse
	enc := json.NewDecoder(req.Body)
	if err := enc.Decode(&robotResp); err != nil {
		return nil, errors.Wrap(err, "error decoding response from DingTalk")
	}

	return &robotResp, nil
}
