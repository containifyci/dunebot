#!/usr/bin/env bash
# Driven asciinema recording of the DuneBot installation walkthrough.
# Each "typed" command is printed char-by-char then executed (where safe).

set -u

# Speed of "typing" (seconds per char)
TYPING_DELAY=0.012
# Pause after a command output before the next command
POST_DELAY=1.2

clear_text() { printf '\033[2J\033[3J\033[H'; }

type_cmd() {
  # $1 = command text to "type"
  local cmd="$1"
  local i ch
  printf '\033[32m$\033[0m '
  for (( i=0; i<${#cmd}; i++ )); do
    ch="${cmd:$i:1}"
    printf '%s' "$ch"
    sleep "$TYPING_DELAY"
  done
  printf '\n'
}

run() {
  # $1 = command to type AND execute
  type_cmd "$1"
  sleep 0.25
  eval "$1" 2>&1 | head -40
  sleep "$POST_DELAY"
}

show() {
  # $1 = command to type but NOT execute (just show output we print)
  type_cmd "$1"
  sleep 0.3
}

print_block() {
  # $1 = text to print verbatim (no typing)
  printf '%s\n' "$1"
  sleep "$POST_DELAY"
}

# ── Title ──────────────────────────────────────────────────────────────────
print_block "$(cat <<'BANNER'

  ╔══════════════════════════════════════════════════════════════╗
  ║          DuneBot — Installation Walkthrough                  ║
  ║   Set up DuneBot for a private GitHub organization           ║
  ╚══════════════════════════════════════════════════════════════╝

  This recording covers the terminal portions of the install:
  1. Clone the repo
  2. Prepare the .env configuration
  3. Deploy with Docker Compose
  4. Connect a code-owner account
  5. Add a dunebot.yaml to a repository
  6. Verify with a test Pull Request

BANNER
)"

# ── Step 1: Clone ──────────────────────────────────────────────────────────
print_block "── Step 1 · Clone the repository ──────────────────────────────────"
run 'git clone https://github.com/containifyci/dunebot.git ~/dunebot'
run 'cd ~/dunebot && ls -1'

# ── Step 2: .env ───────────────────────────────────────────────────────────
print_block "── Step 2 · Prepare environment variables ───────────────────────"
show 'cp .env .env.example    # keep the template'
run 'cp .env .env.example'

print_block "$(cat <<'ENVNOTE'
  Edit .env and fill in the values from your GitHub App:

  DUNEBOT_GITHUB_APP_INTEGRATION_ID   = <App ID>
  DUNEBOT_GITHUB_APP_PRIVATE_KEY      = <base64 of the .pem>
  DUNEBOT_GITHUB_APP_WEBHOOK_SECRET   = <webhook secret>
  DUNEBOT_GITHUB_OAUTH_CLIENT_ID      = <Client ID>
  DUNEBOT_GITHUB_OAUTH_CLIENT_SECRET  = <Client Secret>
  DUNEBOT_APP_CONFIGURATION_INSTALLATION_ID = <installation id>
ENVNOTE
)"

show "base64 -w0 dunebot-private-key.pem   # encode the private key"
run 'echo "# base64 -w0 dunebot-private-key.pem" && echo "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQp...kCk="'

print_block "  Tip: in production, store secrets in a secret manager (Teller, Vault, ..."
run 'grep -c DUNEBOT .env.example | xargs -I{} echo "{} environment variables defined in .env"'

# ── Step 3: Deploy ─────────────────────────────────────────────────────────
print_block "── Step 3 · Deploy with Docker Compose ──────────────────────────"
show 'docker compose up -d --build'
run 'docker compose up -d --build 2>&1 | tail -8'

print_block "  Two services start:"
print_block "    • app   — DuneBot server (webhooks + dashboard) on :8080"
print_block "    • token — JWT token service on :50051 (gRPC) & :8081 (HTTP)"

run 'docker compose ps'

# ── Step 4: Connect code-owner ─────────────────────────────────────────────
print_block "── Step 4 · Connect a code-owner account ────────────────────────"
print_block "  Open in a browser:"
print_block "    https://<your-host>/app/github/setup?installation_id=<id>"
print_block ""
print_block "  DuneBot runs the GitHub OAuth device flow and shows a one-time code:"
run 'echo "  Please go to https://github.com/login/device and enter code:  DUNE-BOTX"'
print_block "  Authorize with a CODEOWNER account → DuneBot stores the token."

# ── Step 5: dunebot.yaml ───────────────────────────────────────────────────
print_block "── Step 5 · Add .github/dunebot.yaml to a repository ────────────"
show 'cat .github/dunebot.yaml'
run 'cat example/dunebot.yml | head -20'

# ── Step 6: Verify ─────────────────────────────────────────────────────────
print_block "── Step 6 · Verify with a test Pull Request ─────────────────────"
show 'docker compose logs -f app'
run 'docker compose logs app 2>&1 | tail -6'

print_block "$(cat <<'OUTRO'
  ─────────────────────────────────────────────────────────────────
  ✅  DuneBot is running! Open a Dependabot PR to see it approve &
     merge automatically when all required statuses pass.

  📖  Full guide:  docs/installation-guide.md
  🎬  Video page:  docs/video-tutorial.md

  Thanks for watching!
OUTRO
)"

sleep 2