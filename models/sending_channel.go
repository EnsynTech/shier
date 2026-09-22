package models

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gophish/gophish/dialer"
	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

const (
	// ChannelEmail is the default SMTP email delivery channel.
	ChannelEmail = "email"
	// ChannelSMS is the generic HTTP SMS gateway channel.
	ChannelSMS = "sms"
	// ChannelIM is the generic HTTP webhook IM channel.
	ChannelIM = "im"

	// EnvAllowHTTPChannels allows non-HTTPS endpoints when set to "1"/"true".
	EnvAllowHTTPChannels = "GOPHISH_ALLOW_HTTP_CHANNELS"
)

// SendingChannel stores a reusable HTTP gateway configuration for SMS or IM.
type SendingChannel struct {
	Id                 int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId             int64     `json:"-" gorm:"column:user_id"`
	Name               string    `json:"name"`
	Channel            string    `json:"channel"`
	EndpointURL        string    `json:"endpoint_url"`
	HTTPMethod         string    `json:"http_method"`
	AuthHeaderName     string    `json:"auth_header_name"`
	AuthHeaderValue    string    `json:"auth_header_value,omitempty"`
	ContentType        string    `json:"content_type"`
	BodyTemplate       string    `json:"body_template"`
	SuccessStatusCodes string    `json:"success_status_codes"`
	IgnoreCertErrors   bool      `json:"ignore_cert_errors"`
	ModifiedDate       time.Time `json:"modified_date"`
}

// DeliveryBodyContext is the template context for rendering gateway request bodies.
type DeliveryBodyContext struct {
	Phone     string
	Email     string
	IMUser    string
	FirstName string
	LastName  string
	Position  string
	URL       string
	RId       string
	Message   string
	Subject   string
}

var (
	// ErrSendingChannelNameNotSpecified is returned when the channel has no name.
	ErrSendingChannelNameNotSpecified = errors.New("Sending channel name not specified")
	// ErrSendingChannelTypeInvalid is returned for unknown channel types.
	ErrSendingChannelTypeInvalid = errors.New("Sending channel must be sms or im")
	// ErrSendingChannelURLNotSpecified is returned when endpoint URL is empty.
	ErrSendingChannelURLNotSpecified = errors.New("Endpoint URL not specified")
	// ErrSendingChannelURLInvalid is returned for non-HTTPS (or disallowed HTTP) URLs.
	ErrSendingChannelURLInvalid = errors.New("Endpoint URL must use https (http only allowed for localhost or when GOPHISH_ALLOW_HTTP_CHANNELS is set)")
	// ErrSendingChannelBodyNotSpecified is returned when the body template is empty.
	ErrSendingChannelBodyNotSpecified = errors.New("Body template not specified")
	// ErrSendingChannelNotFound is returned when a channel cannot be located.
	ErrSendingChannelNotFound = errors.New("Sending channel not found")
	// ErrSendingChannelNotSpecified is returned when a campaign needs a channel but none was given.
	ErrSendingChannelNotSpecified = errors.New("No sending channel specified")
	// ErrPhoneNotSpecified is returned when an SMS campaign target has no phone.
	ErrPhoneNotSpecified = errors.New("Phone number not specified for SMS target")
	// ErrTemplateTextRequired is returned when SMS/IM templates lack text content.
	ErrTemplateTextRequired = errors.New("Message template text is required for SMS/IM channels")
)

// TableName specifies the database table for GORM.
func (SendingChannel) TableName() string {
	return "sending_channels"
}

// Validate ensures the sending channel configuration is usable.
func (s *SendingChannel) Validate() error {
	s.Channel = strings.ToLower(strings.TrimSpace(s.Channel))
	s.HTTPMethod = strings.ToUpper(strings.TrimSpace(s.HTTPMethod))
	s.EndpointURL = strings.TrimSpace(s.EndpointURL)
	s.ContentType = strings.TrimSpace(s.ContentType)
	s.SuccessStatusCodes = strings.TrimSpace(s.SuccessStatusCodes)
	if s.Name == "" {
		return ErrSendingChannelNameNotSpecified
	}
	if s.Channel != ChannelSMS && s.Channel != ChannelIM {
		return ErrSendingChannelTypeInvalid
	}
	if s.EndpointURL == "" {
		return ErrSendingChannelURLNotSpecified
	}
	if err := validateChannelEndpointURL(s.EndpointURL); err != nil {
		return err
	}
	if s.HTTPMethod == "" {
		s.HTTPMethod = http.MethodPost
	}
	switch s.HTTPMethod {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return fmt.Errorf("unsupported HTTP method: %s", s.HTTPMethod)
	}
	if s.ContentType == "" {
		s.ContentType = "application/json"
	}
	if s.BodyTemplate == "" {
		return ErrSendingChannelBodyNotSpecified
	}
	if s.SuccessStatusCodes == "" {
		s.SuccessStatusCodes = "200,201,204"
	}
	if _, err := template.New("body").Parse(s.BodyTemplate); err != nil {
		return fmt.Errorf("invalid body template: %v", err)
	}
	return nil
}

func validateChannelEndpointURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrSendingChannelURLInvalid
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if allowHTTPChannels() || isLoopbackHost(u.Hostname()) {
			return nil
		}
		return ErrSendingChannelURLInvalid
	default:
		return ErrSendingChannelURLInvalid
	}
}

func allowHTTPChannels() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvAllowHTTPChannels)))
	return v == "1" || v == "true" || v == "yes"
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// MaskSecret clears the auth header value for API responses.
func (s *SendingChannel) MaskSecret() {
	if s.AuthHeaderValue != "" {
		s.AuthHeaderValue = ""
	}
}

// GetSendingChannels returns all channels owned by the user.
func GetSendingChannels(uid int64) ([]SendingChannel, error) {
	cs := []SendingChannel{}
	err := db.Where("user_id=?", uid).Find(&cs).Error
	return cs, err
}

// GetSendingChannel returns a channel by id owned by the user.
func GetSendingChannel(id int64, uid int64) (SendingChannel, error) {
	s := SendingChannel{}
	err := db.Where("user_id=? and id=?", uid, id).Find(&s).Error
	if err == gorm.ErrRecordNotFound {
		return s, ErrSendingChannelNotFound
	}
	return s, err
}

// GetSendingChannelByName returns a channel by name owned by the user.
func GetSendingChannelByName(name string, uid int64) (SendingChannel, error) {
	s := SendingChannel{}
	err := db.Where("user_id=? and name=?", uid, name).Find(&s).Error
	if err == gorm.ErrRecordNotFound {
		return s, ErrSendingChannelNotFound
	}
	return s, err
}

// PostSendingChannel creates a new sending channel.
func PostSendingChannel(s *SendingChannel) error {
	if err := s.Validate(); err != nil {
		return err
	}
	s.ModifiedDate = time.Now().UTC()
	return db.Save(s).Error
}

// PutSendingChannel updates an existing sending channel.
func PutSendingChannel(s *SendingChannel) error {
	if err := s.Validate(); err != nil {
		return err
	}
	existing, err := GetSendingChannel(s.Id, s.UserId)
	if err != nil {
		return err
	}
	if s.AuthHeaderValue == "" {
		s.AuthHeaderValue = existing.AuthHeaderValue
	}
	s.ModifiedDate = time.Now().UTC()
	return db.Where("id=?", s.Id).Save(s).Error
}

// DeleteSendingChannel deletes a sending channel by id for the user.
func DeleteSendingChannel(id int64, uid int64) error {
	return db.Where("user_id=? and id=?", uid, id).Delete(&SendingChannel{}).Error
}

// RenderBody renders the body template with the given context.
func (s *SendingChannel) RenderBody(ctx DeliveryBodyContext) (string, error) {
	tmpl, err := template.New("body").Parse(s.BodyTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	body := buf.String()
	if strings.Contains(strings.ToLower(s.ContentType), "json") {
		if !json.Valid([]byte(body)) {
			return "", fmt.Errorf("rendered body is not valid JSON")
		}
	}
	return body, nil
}

// SuccessStatusSet returns the configured HTTP status codes treated as success.
func (s *SendingChannel) SuccessStatusSet() map[int]bool {
	out := make(map[int]bool)
	for _, part := range strings.Split(s.SuccessStatusCodes, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out[code] = true
	}
	if len(out) == 0 {
		out[200] = true
		out[201] = true
		out[204] = true
	}
	return out
}

// Deliver sends one HTTP request using the channel configuration.
func (s *SendingChannel) Deliver(ctx DeliveryBodyContext) error {
	body, err := s.RenderBody(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(s.HTTPMethod, s.EndpointURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	if s.ContentType != "" {
		req.Header.Set("Content-Type", s.ContentType)
	}
	if s.AuthHeaderName != "" && s.AuthHeaderValue != "" {
		req.Header.Set(s.AuthHeaderName, s.AuthHeaderValue)
	}
	transport := &http.Transport{
		DialContext: dialer.Dialer().DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: s.IgnoreCertErrors, // #nosec G402 -- intentional for lab gateways
		},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, io.LimitReader(resp.Body, 1<<20))
	if !s.SuccessStatusSet()[resp.StatusCode] {
		return fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}
	return nil
}

// SendingChannelTestRequest is the payload for a one-off test send.
type SendingChannelTestRequest struct {
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
	URL       string `json:"url"`
}

// TestDeliver performs a test send without creating campaign results.
func (s *SendingChannel) TestDeliver(req SendingChannelTestRequest) error {
	if err := s.Validate(); err != nil {
		return err
	}
	msg := req.Message
	if msg == "" {
		msg = "识饵测试消息 / Shier test message"
	}
	urlVal := req.URL
	if urlVal == "" {
		urlVal = "https://example.com/?rid=test"
	}
	email := req.Email
	if email == "" {
		email = "test@example.com"
	}
	ctx := DeliveryBodyContext{
		Phone:     req.Phone,
		Email:     email,
		IMUser:    email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		URL:       urlVal,
		RId:       "test",
		Message:   msg,
		Subject:   req.Subject,
	}
	if s.Channel == ChannelSMS && ctx.Phone == "" {
		return ErrPhoneNotSpecified
	}
	log.Infof("Sending test %s delivery via channel %s", s.Channel, s.Name)
	return s.Deliver(ctx)
}
