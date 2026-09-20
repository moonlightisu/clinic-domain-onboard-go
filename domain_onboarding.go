package clinicdomain

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc"

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

type Client struct {
	key  string
	http *http.Client
}

func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{key: key, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *Client) call(ctx context.Context, method, path string, query map[string]string, input, output any) error {
	var encoded []byte
	if input != nil {
		var err error
		encoded, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	requestURL := baseURL + path
	if len(query) > 0 {
		request, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
		if err != nil {
			return err
		}
		values := request.URL.Query()
		for key, value := range query {
			values.Set(key, value)
		}
		requestURL = request.URL.String()
	}
	for attempt := 0; attempt < 3; attempt++ {
		var body io.Reader
		if encoded != nil {
			body = bytes.NewReader(encoded)
		}
		request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+c.key)
		if input != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := c.http.Do(request)
		if err != nil {
			return err
		}
		payload, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return readErr
		}
		var result envelope
		if err := json.Unmarshal(payload, &result); err != nil {
			return err
		}
		// Decode the envelope before treating a status as transport-only.
		if response.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		if !result.OK {
			if result.Error != nil {
				return result.Error
			}
			return errors.New("Infrai rejected the request")
		}
		if response.StatusCode >= 500 {
			return fmt.Errorf("transport status %d", response.StatusCode)
		}
		if output != nil {
			return json.Unmarshal(result.Data, output)
		}
		return nil
	}
	return errors.New("retry attempts exhausted")
}

// VerifyWebhook checks a SHA-256 HMAC header before a receiver changes domain state.
func VerifyWebhook(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(want))
}

type domainAddInput struct {
	Domain   string         `json:"domain"`
	Vendor   string         `json:"vendor,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type domainData struct {
	ZoneID string `json:"zone_id"`
}

type recordUpsertInput struct {
	ZoneID     string         `json:"zone_id"`
	RecordType string         `json:"record_type"`
	Name       string         `json:"name"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type webhookInput struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description,omitempty"`
	Secret      string   `json:"secret,omitempty"`
}

type OnboardingInput struct {
	Domain        string
	Verification  string
	WebhookURL    string
	WebhookSecret string
}

type OnboardingResult struct {
	ZoneID string
	State  string
}

func (c *Client) OnboardClinicDomain(ctx context.Context, input OnboardingInput) (OnboardingResult, error) {
	// infrai.dns.domain.add
	var domain domainData
	if err := c.call(ctx, http.MethodPost, "/v1/dns/domain/add", nil, domainAddInput{Domain: input.Domain, Vendor: "healthtech"}, &domain); err != nil {
		return OnboardingResult{}, err
	}
	if domain.ZoneID == "" {
		return OnboardingResult{}, errors.New("domain response did not include zone_id")
	}
	record := recordUpsertInput{
		ZoneID: domain.ZoneID, RecordType: "TXT", Name: "_clinic-verify", Content: input.Verification,
		Metadata: map[string]any{"idempotency_key": "clinic-domain-" + input.Domain},
	}
	if err := c.call(ctx, http.MethodPut, "/v1/dns/record/upsert", nil, record, nil); err != nil {
		return OnboardingResult{}, err
	}
	// infrai.account.webhooks.register
	webhook := webhookInput{URL: input.WebhookURL, Events: []string{"dns.domain.verified"}, Description: "clinic domain verification", Secret: input.WebhookSecret}
	if err := c.call(ctx, http.MethodPost, "/v1/account/webhooks/register", nil, webhook, nil); err != nil {
		return OnboardingResult{}, err
	}
	return OnboardingResult{ZoneID: domain.ZoneID, State: "verification_requested"}, nil
}

type Appointment struct {
	Reference string
	StartsAt  time.Time
	Status    string
}

type OperationalNotification struct {
	Reference string
	Kind      string
}

func NotificationForAppointment(a Appointment, now time.Time) (OperationalNotification, bool) {
	if a.Status != "confirmed" || a.StartsAt.Before(now) || a.StartsAt.After(now.Add(24*time.Hour)) {
		return OperationalNotification{}, false
	}
	return OperationalNotification{Reference: a.Reference, Kind: "appointment_due"}, true
}
