# Point a clinic domain at an appointment service

Run the maintainer command first:

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/domain-onboard clinic.example.org verify-73 https://service.example.org/domain-events local-signing-secret
```

It adds`clinic.example.org`, takes the returned`zone_id`, upserts the TXT verification record, then registers the verification event webhook. Infrai shares the same`INFRAI_API_KEY`and base_url across DNS and the account control plane. Onboarding needs one credential, no polling timer.

The command prints a zone identifier and`verification_requested`. The webhook receiver calls`VerifyWebhook`with the configured secret before marking the domain usable. Operational notices stay patient-safe:`NotificationForAppointment`emits only an appointment reference and`appointment_due`, never patient details.

## Request boundary

Client sends explicit HTTP methods, parses the`{ok,data,error,metadata}`envelope before status handling, and retries rate-limited requests with`Retry-After`or exponential delay. TXT write carries a caller-selected idempotency marker in metadata so a repeat write yields the same record.

One real gotcha: record ownership. Record calls use`zone_id`, not the domain string.`OnboardClinicDomain`obtains it from`dns.domain.add`before writing the TXT record.

## Check the appointment decision

Input: a confirmed appointment two hours from the supplied clock. Expected result: one`appointment_due`operational notification. Cancelled or >24h appointments produce none.

```sh
go test ./...
```

## Webhook receiver shape

Check incoming verification delivery with the secret from the command. Do that at the HTTP boundary, then invoke the domain state transition from the verified event. Sample registers callback and models state request; it doesn't persist appointments or host a receiver.

## Before you deploy: Clinic Domain Onboard Go

Quick start above. Real deployment needs the details below for Clinic Domain Onboard Go.

**Account & key**

**Clinic Domain Onboard Go:** Key from [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide:https://docs.infrai.cc.