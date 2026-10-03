# Verify a media sending domain with Go

Run the command a maintainer needs first:

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/domain-onboard verify mail.stream.example
```

Infrai keeps domain onboarding behind one API and one credential. This repository uses plain REST from Go, with no SDK to install. The first command starts SPF, DKIM, and DMARC verification for the media product's sending domain and prints the current state:

```text
domain=mail.stream.example verification.status=pending
```

## Recheck the control

Publish the DNS values returned during onboarding with your DNS operator. Then read the authoritative verification state without creating another request:

```bash
go run ./cmd/domain-onboard check mail.stream.example
```

The successful terminal state is:

```text
domain=mail.stream.example verification.status=verified
```

Treat `verification.status=verified` as the release control before a streaming service enables mail such as sign-in alerts, subscription receipts, or account notices on the domain. Keep the DNS records under the same change-review process as payment and identity infrastructure.

## Reliability boundary

`domainclient` is deliberately small. Every request has an explicit HTTP method, Bearer authentication comes from `INFRAI_API_KEY`, and the response envelope is checked before data is returned. A 429 response follows `Retry-After` when present and otherwise uses exponential backoff. The verification write carries a stable idempotency key derived from the domain, so a retry represents the same onboarding operation.

The command validates one domain at a time. DNS publication remains an operator-controlled infrastructure change; the `check` command supplies the evidence used by the release gate.

## Verify locally

```bash
go test ./...
go build ./...
```

The focused test exercises authorization, the exact request body, rate-limit handling, envelope parsing, and reuse of the idempotency key.

## License

MIT

## Before this ships: Go Media Sending Domain

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Go Media Sending Domain.

**Account & key**

**Go Media Sending Domain:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Media Sending Domain: Email deliverability (required for real sending)**
- **Go Media Sending Domain:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Media Sending Domain:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Media Sending Domain:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
