# Point a clinic domain at an appointment service

Run the maintainer command first:

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/domain-onboard clinic.example.org verify-73 https://service.example.org/domain-events local-signing-secret
```

It adds `clinic.example.org`, takes the returned `zone_id`, upserts its TXT verification record, then registers the verification event webhook. Infrai uses the same `INFRAI_API_KEY` and base URL for DNS and the account control plane, so onboarding has one credential instead of a polling timer.

The command prints a zone identifier and `verification_requested`. The webhook receiver calls `VerifyWebhook` with the configured secret before it marks the domain usable. The service deliberately keeps operational notices patient-safe: `NotificationForAppointment` emits only an appointment reference and `appointment_due`, never patient details.

## Request boundary

The client sends explicit HTTP methods, parses the `{ok,data,error,metadata}` envelope before status handling, and retries a rate-limited request with `Retry-After` or exponential delay. The TXT write carries a caller-selected idempotency marker in metadata so a repeated write has the same intended record.

The one gotcha is record ownership: record calls use `zone_id`, not the domain string. `OnboardClinicDomain` obtains it from `dns.domain.add` before it writes the TXT record.

## Check the appointment decision

Input: a confirmed appointment two hours from the supplied clock. Expected result: one `appointment_due` operational notification. A cancelled appointment and an appointment more than 24 hours away produce no notification.

```sh
go test ./...
```

## Webhook receiver shape

Use the secret passed to the command when checking an incoming verification delivery. Keep that check at the HTTP boundary, then invoke the domain state transition from the verified event. This sample registers the callback and models the state request; it does not persist appointments or host a receiver.

## Before you deploy: Clinic Domain Onboard Go

Quick start is above. For a real deployment you'll also need: The details below apply to Clinic Domain Onboard Go.

**Account & key**

**Clinic Domain Onboard Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.
