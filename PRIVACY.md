# Privacy Policy for DuneBot

DuneBot is a GitHub App that automates pull request workflows. This document describes how data is handled.

# Data Collection

DuneBot processes data provided by GitHub webhooks, including:

* Repository metadata (names, branches)
* Pull request data (titles, descriptions, status checks)
* Commit metadata

# Data Usage

Collected data is used solely to:

* Evaluate pull request rules
* Approve, rebase, or merge pull requests
* Execute configured automation workflows

# Data Storage

No long-term storage of repository or pull request data is performed
All data is processed in memory during execution only
DuneBot uses only the GitHub App installation token, which is stored temporarily in memory while processing requests
Tokens and request data are not persisted to disk
Configuration is read from repository files (.github/dunebot.yaml)

DuneBot does not share, sell, or transfer data to third parties.

# Security

Reasonable measures are taken to protect data and credentials. However, as a personal project, no guarantees of security or availability are provided.

# Third-Party Services

DuneBot operates on infrastructure hosted by Hetzner Cloud and depends on GitHub APIs.

# Changes

This policy may be updated at any time without prior notice.

# Contact

For questions, please open an issue in the GitHub repository.