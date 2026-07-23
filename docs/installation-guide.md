# DuneBot Installation Guide

This guide walks you through installing and configuring **DuneBot** for a private
GitHub organization from scratch. It is the written companion to the
[Video Tutorial](./video-tutorial.md).

DuneBot is a self-hosted GitHub App that automatically approves and merges Pull
Requests (the primary use case is Dependabot PRs). To run it you need to:

1. Create a GitHub App
2. Configure the OAuth credentials
3. Deploy the DuneBot server (Docker Compose or binary)
4. Install the App on your organization
5. Connect a GitHub account with code-ownership rights
6. Add a `dunebot.yaml` to your repositories

---

## Prerequisites

| Requirement              | Notes                                                                 |
|--------------------------|-----------------------------------------------------------------------|
| A GitHub account         | Owner permission on the target organization                           |
| A host / server          | Any machine reachable from GitHub webhooks (public URL or tunnel)     |
| Docker & Docker Compose  | Recommended deployment path                                           |
| `git`                    | To clone the repository                                               |
| A domain / public URL    | GitHub must be able to reach `https://<your-host>` for webhooks & OAuth |

> **Why a public URL?** GitHub sends webhook events to your DuneBot instance and
> redirects the OAuth flow back to it. Both require a publicly reachable endpoint.

---

## Step 1 — Create the GitHub App

1. Go to **GitHub → Settings → Developer settings → GitHub Apps → New GitHub App**
   (for an organization app, open the org's **Settings → Developer settings →
   GitHub Apps → New GitHub App**).
2. Fill in the form:

   | Field                          | Value                                            |
   |--------------------------------|--------------------------------------------------|
   | **GitHub App name**            | `DuneBot` (or `<Org> DuneBot`)                   |
   | **Homepage URL**               | `https://<your-host>`                            |
   | **Callback URL**               | `https://<your-host>/api/github/auth`            |
   | **Webhook URL**                | `https://<your-host>/api/github/hook`            |
   | **Webhook secret**             | a strong random string — **save it**             |

3. **Repository permissions** — grant the following:

   | Permission            | Access    |
   |-----------------------|-----------|
   | Pull requests         | **Read & write** |
   | Contents              | **Read & write** |
   | Commit statuses       | Read-only |
   | Metadata              | Read-only |
   | Issues                | Read-only |

4. **Organization permissions** — grant:

   | Permission                | Access    |
   |---------------------------|-----------|
   | Members                   | Read-only |
   | Administration            | Read-only |

5. **Subscribe to events**: `Pull request`, `Pull request review`, `Issue comment`.
6. Click **Create GitHub App**.

7. After creation:
   - Note the **App ID** (`DUNEBOT_GITHUB_APP_INTEGRATION_ID`).
   - Generate and download a **private key** `.pem` file → this becomes
     `DUNEBOT_GITHUB_APP_PRIVATE_KEY` (base64-encoded, see Step 3).
   - Note the **Client ID** (`DUNEBOT_GITHUB_OAUTH_CLIENT_ID`).
   - Generate a **Client Secret** → `DUNEBOT_GITHUB_OAUTH_CLIENT_SECRET`.

> Keep the private key and secrets safe — you will inject them as environment
> variables in Step 3.

---

## Step 2 — Install the App on your Organization

1. On the App's page, click **Install App** in the left sidebar.
2. Choose the organization you want DuneBot to manage.
3. Select **All repositories** or pick specific repositories.
4. Click **Install**.

GitHub will redirect to your **Callback URL** with an `installation_id` query
parameter — this is your `DUNEBOT_APP_CONFIGURATION_INSTALLATION_ID`.
If you miss it, you can find the installation id under
**Settings → Integrations → GitHub Apps → Configure** for the org, or by
listing installations via the GitHub API.

---

## Step 3 — Prepare Environment Variables

DuneBot is configured entirely through environment variables (see `.env` for the
full reference). Create a `.env` file on your host:

```bash
# --- GitHub App credentials (from Step 1) ---
DUNEBOT_GITHUB_APP_INTEGRATION_ID=<App ID>
# base64-encode the downloaded .pem private key:
#   base64 -w0 dunebot-private-key.pem
DUNEBOT_GITHUB_APP_PRIVATE_KEY=<base64-encoded private key>
DUNEBOT_GITHUB_APP_WEBHOOK_SECRET=<webhook secret from Step 1>

# --- OAuth credentials (from Step 1) ---
DUNEBOT_GITHUB_OAUTH_CLIENT_ID=<Client ID>
DUNEBOT_GITHUB_OAUTH_CLIENT_SECRET=<Client Secret>
DUNEBOT_GITHUB_OAUTH_REDIRECT_URL=https://<your-host>/api/github/auth
DUNEBOT_GITHUB_OAUTH_SCOPES=repo

# --- Installation (from Step 2) ---
DUNEBOT_APP_CONFIGURATION_INSTALLATION_ID=<installation id>

# --- Server ---
DUNEBOT_SERVER_ADDRESS=0.0.0.0
DUNEBOT_SERVER_PORT=8080

# --- JWT (optional, enables token service) ---
# Generate with: openssl ecparam -genkey -name prime256v1 -noout -out jwt.pem
# DUNEBOT_JWT_PRIVATE_KEY=<base64 of jwt.pem>
# DUNEBOT_JWT_PUBLIC_KEY=<base64 of the public key>
DUNEBOT_JWT_SERVER_ADDRESS=localhost:50051
```

> **Secret management**: in production, store these in a secret manager (the
> project uses [Teller](https://tlr.dev) with Google Secret Manager — see
> `.teller.yml`). For a quick start, a plain `.env` file behind a reverse proxy
> with TLS is sufficient.

---

## Step 4 — Deploy DuneBot

### Option A — Docker Compose (recommended)

```bash
git clone https://github.com/containifyci/dunebot.git
cd dunebot

# place your .env file in the repo root, then:
docker compose up -d --build
```

This starts two services (see `docker-compose.yaml`):

- **app** — the DuneBot server (webhooks + dashboard + API) on port `8080`.
- **token** — the JWT token service on ports `50051` (gRPC) and `8081` (HTTP),
  only needed if you enabled JWT auth.

### Option B — Pre-built binary / Docker image

```bash
# Pull the published image (see the Releases page for the latest tag)
docker run -d --env-file .env -p 8080:8080 ghcr.io/containifyci/dunebot:latest app
```

### Option C — Build from source

```bash
make build           # produces build/dunebot
env $(cat .env) ./build/dunebot app
```

> Whichever option you choose, put DuneBot behind a TLS-terminating reverse
> proxy (Caddy, nginx, Traefik). GitHub requires HTTPS for webhook URLs.

---

## Step 5 — Connect a Code-Owner Account

After installation, DuneBot needs a GitHub account with **code-ownership**
permissions so it can approve PRs on behalf of a real reviewer.

1. Open `https://<your-host>/app/github/setup?installation_id=<installation_id>`
   in your browser.
2. DuneBot starts the **GitHub OAuth device flow** and displays a user code plus
   a verification link.
3. Click the link, enter the code, and authorize with the account that should act
   as the approver (e.g. a dedicated bot user that is a CODEOWNER).
4. DuneBot stores the resulting token and will use it to approve & merge PRs.

---

## Step 6 — Configure your Repositories

Add a `.github/dunebot.yaml` to each repository DuneBot should manage. A minimal
configuration that auto-approves & squash-merges Dependabot PRs:

```yaml
version: 1.0.0

approve:
  approver: "your-bot-username"
  include:
    - authors:
      - "dependabot[bot]"
  required_statuses:
    - "ci/circleci: build"

merge:
  method: squash
  include:
    - authors:
      - "dependabot[bot]"
  required_statuses:
    - "ci/circleci: build"
```

See [`example/dunebot.yml`](../example/dunebot.yml) for the full set of options
(labels, branches, comments, excludes, etc.).

---

## Step 7 — Verify the Setup

1. Open a test Pull Request in one of the configured repositories (e.g. a
   Dependabot PR, or a PR labeled `approve`).
2. Watch the DuneBot logs: `docker compose logs -f app`.
3. DuneBot should evaluate the rules and, when all `required_statuses` pass,
   approve and merge the PR.

If nothing happens, check that:

- The webhook URL is reachable from GitHub (GitHub → App settings → Advanced →
  Recent Deliveries shows the delivery status & response).
- The environment variables match the values from Steps 1–2.
- The repository has a valid `.github/dunebot.yaml`.

---

## Troubleshooting

| Symptom                                   | Likely cause / fix                                                                 |
|-------------------------------------------|------------------------------------------------------------------------------------|
| Webhook deliveries show `502`/`503`       | DuneBot not running or reverse proxy mis-routed. Check `docker compose ps`.        |
| `401 Unauthorized` on `/api/github/hook`  | `DUNEBOT_GITHUB_APP_WEBHOOK_SECRET` doesn't match the App's webhook secret.        |
| PRs are not approved                      | The code-owner account from Step 5 isn't a CODEOWNER, or rules in `dunebot.yaml` exclude the PR. |
| OAuth redirect fails                      | `DUNEBOT_GITHUB_OAUTH_REDIRECT_URL` must exactly match the App's **Callback URL**. |
| `installation_id` not found               | Re-install the App on the org (Step 2) and copy the id from the redirect URL.      |

---

## Next Steps

- Read the full [`example/dunebot.yml`](../example/dunebot.yml) for advanced rules.
- Watch the [Video Tutorial](./video-tutorial.md) for a walkthrough of every step above.
- Review the [Privacy Policy](../PRIVACY.md) to understand what data DuneBot processes.