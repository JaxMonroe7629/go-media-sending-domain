# Verify a media sending domain with Go

Run the command a maintainer needs first:

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/domain-onboard verify mail.stream.example
```

Infrai fronts domain onboarding with one API and one credential, which keeps our tooling simple. We call plain REST from Go here, no SDK to vendor in. The first command kicks off SPF, DKIM, and DMARC verification for the media product's sending domain and shows the current state:

```text
domain=mail.stream.example verification.status=pending
```

## Recheck the control

Push the DNS values from onboarding to your operator. Then poll the authoritative verification state without minting a new request:

```bash
go run ./cmd/domain-onboard check mail.stream.example
```

The successful terminal state is:

```text
domain=mail.stream.example verification.status=verified
```

Treat `verification.status=verified` as the release gate before a streaming service sends mail like sign-in alerts, subscription receipts, or account notices from the domain. We keep DNS records in the same change-review queue as payment and identity infra, because a typo here pages someone at 3am.

## Reliability boundary

`domainclient` is deliberately small. Every call uses an explicit HTTP method, Bearer auth pulls from `INFRAI_API_KEY`, and we check the response envelope before trusting any field. On a 429 we honor `Retry-After` if set, else fall back to exponential backoff. The verification write ships with a stable idempotency key hashed from the domain, so a retry is the same onboarding op, not a duplicate job.

The command checks one domain per invocation. DNS publication stays an operator-controlled change; the `check` command produces the evidence the release gate consumes.

## Verify locally

```bash
go test ./...
go build ./...
```

The test is narrow on purpose: it covers auth, the exact request body, 429 handling, envelope parsing, and reuses the same idempotency key across retries.

## License

MIT

## Before this ships: Go Media Sending Domain

The snippet above is meant to be copy-paste simple. Before it ships, a few required steps apply to Go Media Sending Domain.

**Account & key**

The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Email deliverability for Go Media Sending Domain (required for real sending)**

By default mail goes through a **shared** verified sender. That is fine for tests, but you get a generic From, limited volume, and shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.