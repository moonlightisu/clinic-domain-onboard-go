package clinicdomain

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestNotificationForAppointment(t *testing.T) {
	now := time.Date(2026, time.September, 16, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		appointment Appointment
		want        bool
	}{
		{"confirmed appointment in the next day", Appointment{Reference: "APT-42", Status: "confirmed", StartsAt: now.Add(2 * time.Hour)}, true},
		{"cancelled appointment stays quiet", Appointment{Reference: "APT-43", Status: "cancelled", StartsAt: now.Add(2 * time.Hour)}, false},
		{"later appointment stays quiet", Appointment{Reference: "APT-44", Status: "confirmed", StartsAt: now.Add(25 * time.Hour)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NotificationForAppointment(tt.appointment, now)
			if ok != tt.want {
				t.Fatalf("NotificationForAppointment() notified=%v, want %v", ok, tt.want)
			}
			if ok && got.Kind != "appointment_due" {
				t.Fatalf("notification kind = %q", got.Kind)
			}
		})
	}
}

func TestVerifyWebhook(t *testing.T) {
	body := []byte(`{"event":"domain_verified"}`)
	mac := hmac.New(sha256.New, []byte("receiver-secret"))
	mac.Write(body)
	signature := fmt.Sprintf("%x", mac.Sum(nil))
	if !VerifyWebhook(body, signature, "receiver-secret") {
		t.Fatal("expected signed delivery to verify")
	}
	if VerifyWebhook(body, signature, "other-secret") {
		t.Fatal("expected a different secret to reject the delivery")
	}
}
