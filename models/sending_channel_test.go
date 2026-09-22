package models

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"gopkg.in/check.v1"
)

func (s *ModelsSuite) TestSendingChannelValidate(c *check.C) {
	ch := SendingChannel{
		Name:         "sms-gateway",
		Channel:      ChannelSMS,
		EndpointURL:  "https://sms.example.com/send",
		BodyTemplate: `{"to":"{{.Phone}}","text":"{{.Message}}"}`,
	}
	c.Assert(ch.Validate(), check.IsNil)
	c.Assert(ch.HTTPMethod, check.Equals, http.MethodPost)
	c.Assert(ch.ContentType, check.Equals, "application/json")
}

func (s *ModelsSuite) TestSendingChannelRejectsPlainHTTP(c *check.C) {
	ch := SendingChannel{
		Name:         "bad",
		Channel:      ChannelSMS,
		EndpointURL:  "http://sms.example.com/send",
		BodyTemplate: `{"to":"{{.Phone}}"}`,
	}
	c.Assert(ch.Validate(), check.Equals, ErrSendingChannelURLInvalid)
}

func (s *ModelsSuite) TestSendingChannelAllowsLocalhostHTTP(c *check.C) {
	ch := SendingChannel{
		Name:         "local",
		Channel:      ChannelIM,
		EndpointURL:  "http://127.0.0.1:9/hook",
		BodyTemplate: `{"text":"{{.Message}}"}`,
	}
	c.Assert(ch.Validate(), check.IsNil)
}

func (s *ModelsSuite) TestSendingChannelRenderBodyRequiresJSON(c *check.C) {
	ch := SendingChannel{
		Name:         "sms",
		Channel:      ChannelSMS,
		EndpointURL:  "https://sms.example.com/send",
		ContentType:  "application/json",
		BodyTemplate: `{"to":"{{.Phone}}","text":"{{.Message}}"}`,
	}
	c.Assert(ch.Validate(), check.IsNil)
	body, err := ch.RenderBody(DeliveryBodyContext{
		Phone:   "+8613800138000",
		Message: "hello",
	})
	c.Assert(err, check.IsNil)
	c.Assert(json.Valid([]byte(body)), check.Equals, true)
	c.Assert(strings.Contains(body, "+8613800138000"), check.Equals, true)

	ch.BodyTemplate = `not-json {{.Phone}}`
	_, err = ch.RenderBody(DeliveryBodyContext{Phone: "1"})
	c.Assert(err, check.NotNil)
}

func (s *ModelsSuite) TestSendingChannelDeliverSuccess(c *check.C) {
	var gotMethod, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		b, _ := ioutil.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	os.Setenv(EnvAllowHTTPChannels, "1")
	defer os.Unsetenv(EnvAllowHTTPChannels)

	ch := SendingChannel{
		Name:            "gw",
		Channel:         ChannelSMS,
		EndpointURL:     srv.URL,
		HTTPMethod:      http.MethodPost,
		ContentType:     "application/json",
		AuthHeaderName:  "Authorization",
		AuthHeaderValue: "Bearer test-token",
		BodyTemplate:    `{"phone":"{{.Phone}}","msg":"{{.Message}}","url":"{{.URL}}"}`,
		SuccessStatusCodes: "200",
	}
	c.Assert(ch.Validate(), check.IsNil)
	err := ch.Deliver(DeliveryBodyContext{
		Phone:   "13800138000",
		Message: "click me",
		URL:     "https://phish.example/?rid=abc",
	})
	c.Assert(err, check.IsNil)
	c.Assert(gotMethod, check.Equals, http.MethodPost)
	c.Assert(gotAuth, check.Equals, "Bearer test-token")
	c.Assert(strings.Contains(gotBody, "13800138000"), check.Equals, true)
	c.Assert(strings.Contains(gotBody, "click me"), check.Equals, true)
}

func (s *ModelsSuite) TestSendingChannelCRUD(c *check.C) {
	ch := SendingChannel{
		UserId:       1,
		Name:         "im-bot",
		Channel:      ChannelIM,
		EndpointURL:  "https://hooks.example.com/bot",
		BodyTemplate: `{"msgtype":"text","text":{"content":"{{.Message}} {{.URL}}"}}`,
	}
	c.Assert(PostSendingChannel(&ch), check.IsNil)
	c.Assert(ch.Id, check.Not(check.Equals), int64(0))

	got, err := GetSendingChannel(ch.Id, 1)
	c.Assert(err, check.IsNil)
	c.Assert(got.Name, check.Equals, "im-bot")
	c.Assert(got.AuthHeaderValue, check.Equals, "")

	ch.AuthHeaderName = "X-Token"
	ch.AuthHeaderValue = "secret"
	c.Assert(PutSendingChannel(&ch), check.IsNil)
	got, err = GetSendingChannel(ch.Id, 1)
	c.Assert(err, check.IsNil)
	c.Assert(got.AuthHeaderValue, check.Equals, "secret")

	// Empty auth on update keeps previous secret.
	ch.AuthHeaderValue = ""
	c.Assert(PutSendingChannel(&ch), check.IsNil)
	got, err = GetSendingChannel(ch.Id, 1)
	c.Assert(err, check.IsNil)
	c.Assert(got.AuthHeaderValue, check.Equals, "secret")

	c.Assert(DeleteSendingChannel(ch.Id, 1), check.IsNil)
}

func (s *ModelsSuite) TestCampaignSMSCreatesDeliveryLog(c *check.C) {
	os.Setenv(EnvAllowHTTPChannels, "1")
	defer os.Unsetenv(EnvAllowHTTPChannels)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	ch := SendingChannel{
		UserId:             1,
		Name:               "sms-test-ch",
		Channel:            ChannelSMS,
		EndpointURL:        srv.URL,
		BodyTemplate:       `{"to":"{{.Phone}}","text":"{{.Message}}"}`,
		SuccessStatusCodes: "201",
	}
	c.Assert(PostSendingChannel(&ch), check.IsNil)

	g := Group{
		Name:   "sms-group",
		UserId: 1,
		Targets: []Target{{
			BaseRecipient: BaseRecipient{
				Email:     "smsuser@example.com",
				FirstName: "Sms",
				LastName:  "User",
				Phone:     "13900001111",
			},
		}},
	}
	c.Assert(PostGroup(&g), check.IsNil)

	tmpl := Template{
		Name:   "sms-tmpl",
		UserId: 1,
		Text:   "请点击 {{.URL}}",
	}
	c.Assert(PostTemplate(&tmpl), check.IsNil)
	page := Page{Name: "sms-page", UserId: 1, HTML: "<html>ok</html>"}
	c.Assert(PostPage(&page), check.IsNil)

	campaign := Campaign{
		Name:    "sms-campaign",
		UserId:  1,
		Channel: ChannelSMS,
		Template: Template{Name: tmpl.Name},
		Page:     Page{Name: page.Name},
		URL:      "http://localhost",
		Groups:   []Group{{Name: g.Name}},
		SendingChannel: SendingChannel{Name: ch.Name},
	}
	c.Assert(PostCampaign(&campaign, 1), check.IsNil)
	c.Assert(campaign.Channel, check.Equals, ChannelSMS)
	c.Assert(campaign.SendingChannelId, check.Equals, ch.Id)

	ds, err := GetDeliveryLogsByCampaign(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(len(ds), check.Equals, 1)
	c.Assert(ds[0].Channel, check.Equals, ChannelSMS)

	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(len(ms), check.Equals, 0)

	// Process delivery
	d := ds[0]
	ctx, err := GetCampaignDeliveryContext(campaign.Id, 1)
	c.Assert(err, check.IsNil)
	c.Assert(d.CacheCampaign(&ctx), check.IsNil)
	c.Assert(d.Process(), check.IsNil)

	r, err := GetResult(d.RId)
	c.Assert(err, check.IsNil)
	c.Assert(r.Status, check.Equals, EventSent)
	c.Assert(r.Phone, check.Equals, "13900001111")
}

func (s *ModelsSuite) TestCampaignSMSRequiresPhone(c *check.C) {
	ch := SendingChannel{
		UserId:       1,
		Name:         "sms-nophone",
		Channel:      ChannelSMS,
		EndpointURL:  "https://sms.example.com/x",
		BodyTemplate: `{"to":"{{.Phone}}"}`,
	}
	c.Assert(PostSendingChannel(&ch), check.IsNil)

	g := Group{
		Name:   "sms-nophone-group",
		UserId: 1,
		Targets: []Target{{
			BaseRecipient: BaseRecipient{Email: "nophone@example.com"},
		}},
	}
	c.Assert(PostGroup(&g), check.IsNil)
	tmpl := Template{Name: "sms-nophone-tmpl", UserId: 1, Text: "x {{.URL}}"}
	c.Assert(PostTemplate(&tmpl), check.IsNil)
	page := Page{Name: "sms-nophone-page", UserId: 1, HTML: "<html/>"}
	c.Assert(PostPage(&page), check.IsNil)

	campaign := Campaign{
		Name:           "sms-nophone-camp",
		Channel:        ChannelSMS,
		Template:       Template{Name: tmpl.Name},
		Page:           Page{Name: page.Name},
		URL:            "http://localhost",
		Groups:         []Group{{Name: g.Name}},
		SendingChannel: SendingChannel{Name: ch.Name},
	}
	err := PostCampaign(&campaign, 1)
	c.Assert(err, check.Equals, ErrPhoneNotSpecified)
}

func (s *ModelsSuite) TestScenarioPacksIncludeSMSAndIM(c *check.C) {
	packs, err := ListScenarioPacks()
	c.Assert(err, check.IsNil)
	foundSMS, foundIM := false, false
	for _, p := range packs {
		if p.ID == "sms_parcel_pickup" {
			foundSMS = true
		}
		if p.ID == "im_meeting_card" {
			foundIM = true
		}
	}
	c.Assert(foundSMS, check.Equals, true)
	c.Assert(foundIM, check.Equals, true)
}
