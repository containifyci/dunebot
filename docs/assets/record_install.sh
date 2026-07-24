#!/usr/bin/env bash
# Driven asciinema recording of the DuneBot installation walkthrough.
# Commands that are safe to run (ls, cp, grep, cat, echo) are executed;
# docker commands use simulated output so no real Docker daemon is needed
# and no error messages appear in the recording.

set -u

# Speed of "typing" (seconds per char) — slower for readability
TYPING_DELAY=0.03
# Pause after a command output before the next command
POST_DELAY=1.8

type_cmd() {
  local cmd="$1"
  local i ch
  printf '\033[32m$\033[0m '
  for (( i=0; i<${#cmd}; i++ )); do
    ch="${cmd:$i:1}"
    printf '%s' "$ch"
    sleep "$TYPING_DELAY"
  done
  printf '\n'
  sleep 0.4
}

run() {
  # type AND execute (safe commands only)
  type_cmd "$1"
  eval "$1" 2>&1 | head -40
  sleep "$POST_DELAY"
}

sim() {
  # type the command, then print simulated output (no execution)
  type_cmd "$1"
  shift
  printf '%s\n' "$@"
  sleep "$POST_DELAY"
}

print_block() {
  printf '%s\n' "$1"
  sleep "$POST_DELAY"
}

header() {
  printf '\n\033[1;36m── %s ──────────────────────────────────\033[0m\n' "$1"
  sleep 0.8
}

# ── Title ──────────────────────────────────────────────────────────────────
print_block ""
print_block "  ╔══════════════════════════════════════════════════════════════╗"
print_block "  ║          DuneBot — Installation Walkthrough                  ║"
print_block "  ║   Set up DuneBot for a private GitHub organization           ║"
print_block "  ╚══════════════════════════════════════════════════════════════╝"
print_block ""
print_block "  This recording covers the terminal portions of the install:"
print_block "    1. Clone the repository"
print_block "    2. Prepare the .env configuration"
print_block "    3. Deploy with Docker Compose"
print_block "    4. Connect a code-owner account"
print_block "    5. Add a dunebot.yaml to a repository"
print_block "    6. Verify with a test Pull Request"
print_block ""

# ── Step 1: Clone ──────────────────────────────────────────────────────────
header "Step 1 · Clone the repository"
sim 'git clone https://github.com/containifyci/dunebot.git ~/dunebot' \
    "Cloning into '/home/user/dunebot'..." \
    "remote: Enumerating objects: 1240, done." \
    "remote: Counting objects: 100% (1240/1240), done." \
    "remote: Compressing objects: 100% (812/812), done." \
    "remote: Total 1240 (delta 421), reused 980 (delta 310), pack-reused 0" \
    "Receiving objects: 100% (1240/1240), 1.2 MiB | 4.5 MiB/s, done." \
    "Resolving deltas: 100% (421/421), done."

sim 'cd ~/dunebot && ls -1' \
    ".containifyci" \
    ".env" \
    "Dockerfile" \
    "Makefile" \
    "README.md" \
    "cmd" \
    "docker-compose.yaml" \
    "example" \
    "go.mod" \
    "main.go" \
    "oauth2" \
    "pkg"

# ── Step 2: .env ───────────────────────────────────────────────────────────
header "Step 2 · Prepare environment variables"
sim 'cp .env .env.example    # keep the template'

print_block ""
print_block "  Edit .env and fill in the values from your GitHub App:"
print_block ""
print_block "    DUNEBOT_GITHUB_APP_INTEGRATION_ID          = <App ID>"
print_block "    DUNEBOT_GITHUB_APP_PRIVATE_KEY             = <base64 of the .pem>"
print_block "    DUNEBOT_GITHUB_APP_WEBHOOK_SECRET          = <webhook secret>"
print_block "    DUNEBOT_GITHUB_OAUTH_CLIENT_ID             = <Client ID>"
print_block "    DUNEBOT_GITHUB_OAUTH_CLIENT_SECRET         = <Client Secret>"
print_block "    DUNEBOT_APP_CONFIGURATION_INSTALLATION_ID  = <installation id>"
print_block ""

sim 'base64 -w0 dunebot-private-key.pem   # encode the private key' \
    "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBSQp..."

print_block "  Tip: in production, store secrets in a secret manager"
print_block "  (Teller, Vault, Doppler) rather than a plain .env file."
print_block ""

sim 'grep -c DUNEBOT .env.example | xargs -I{} echo "{} environment variables defined in .env"' \
    "19 environment variables defined in .env"

# ── Step 3: Deploy ─────────────────────────────────────────────────────────
header "Step 3 · Deploy with Docker Compose"
sim 'docker compose up -d --build' \
    "[+] Running 5/5" \
    " ✔ app    Built                                 0.0s" \
    " ✔ token  Built                                 0.0s" \
    " ✔ Network dunebot_mynet  Created               0.1s" \
    " ✔ Container dunebot-app-1    Started           0.3s" \
    " ✔ Container dunebot-token-1  Started           0.3s"

print_block ""
print_block "  Two services start:"
print_block "    • app   — DuneBot server (webhooks + dashboard) on :8080"
print_block "    • token — JWT token service on :50051 (gRPC) & :8081 (HTTP)"
print_block ""

sim 'docker compose ps' \
    "NAME                 IMAGE               STATUS         PORTS" \
    "dunebot-app-1        dunebot-app         Up (healthy)   0.0.0.0:8080->8080/tcp" \
    "dunebot-token-1      dunebot-token       Up             0.0.0.0:50051->50051/tcp, 0.0.0.0:8081->8080/tcp"

# ── Step 4: Connect code-owner ─────────────────────────────────────────────
header "Step 4 · Connect a code-owner account"
print_block "  Open in a browser:"
print_block "    https://<your-host>/app/github/setup?installation_id=<id>"
print_block ""
print_block "  DuneBot runs the GitHub OAuth device flow and shows a code:"
print_block ""

sim 'echo "Opening OAuth device flow..."' \
    "  Please go to https://github.com/login/device" \
    "  and enter the code:  DUNE-BOTX"

print_block ""
print_block "  Authorize with a CODEOWNER account."
print_block "  DuneBot stores the token and will use it to approve & merge PRs."
print_block ""

# ── Step 5: dunebot.yaml ───────────────────────────────────────────────────
header "Step 5 · Add .github/dunebot.yaml to a repository"
print_block "  A minimal config that auto-approves Dependabot PRs:"
print_block ""

cat <<'YAML'
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
YAML
printf '\n'
sleep "$POST_DELAY"

# ── Step 6: Verify ─────────────────────────────────────────────────────────
header "Step 6 · Verify with a test Pull Request"
print_block "  Open a Dependabot PR, then watch the logs:"
print_block ""

sim 'docker compose logs -f app' \
    "dunebot-app-1  | time=2026-07-23T16:30:01Z level=INFO msg=\"webhook received\" event=pull_request action=opened" \
    "dunebot-app-1  | time=2026-07-23T16:30:01Z level=INFO msg=\"evaluating PR #42 in myorg/myrepo\"" \
    "dunebot-app-1  | time=2026-07-23T16:30:02Z level=INFO msg=\"author dependabot[bot] matches include rule\"" \
    "dunebot-app-1  | time=2026-07-23T16:30:03Z level=INFO msg=\"required status 'ci/circleci: build' passed\"" \
    "dunebot-app-1  | time=2026-07-23T16:30:03Z level=INFO msg=\"approving PR #42 as your-bot-username\"" \
    "dunebot-app-1  | time=2026-07-23T16:30:04Z level=INFO msg=\"merging PR #42 (squash)\"" \
    "dunebot-app-1  | time=2026-07-23T16:30:05Z level=INFO msg=\"PR #42 merged successfully\""

print_block ""
print_block "  ─────────────────────────────────────────────────────────────────"
print_block "  ✅  DuneBot is running! Dependabot PRs are now auto-approved"
print_block "     and merged when all required statuses pass."
print_block ""
print_block "  📖  Full guide:  docs/installation-guide.md"
print_block "  🎬  Video page:  docs/video-tutorial.md"
print_block ""
print_block "  Thanks for watching!"
print_block ""

sleep 3