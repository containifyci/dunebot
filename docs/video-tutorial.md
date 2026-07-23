# DuneBot Video Tutorial

A short video walkthrough demonstrating how to set up DuneBot for a private
GitHub organization, end to end.

> 🎬 **Video status:** _Recording planned — see the outline below._
> Once published, the video will be embedded here and linked from the README.
> If you'd like to record it, the script/outline below covers every step.

<!-- Replace the placeholder below with the embedded video once it is published. -->
<!-- Example: <a href="https://youtu.be/XXXXXXXXXXX"><img src="docs/assets/dunebot-tutorial-thumbnail.png" width="480" alt="DuneBot installation tutorial"/></a> -->

[![DuneBot Installation Tutorial — placeholder](https://img.youtube.com/vi/dQw4w9WgXcQ/0.jpg)](#video-placeholder)

---

## Video Outline

The video follows the [Installation Guide](./installation-guide.md) step by step.
Estimated length: **~8 minutes**.

| #  | Section                              | Duration | What is shown                                                                 |
|----|--------------------------------------|----------|-------------------------------------------------------------------------------|
| 1  | Intro & overview                     | 0:30     | What DuneBot does, the architecture diagram, what you'll have at the end.     |
| 2  | Prerequisites                        | 0:20     | A GitHub org with owner rights, a host with a public URL, Docker installed.   |
| 3  | Create the GitHub App                | 2:00     | Screen-recording GitHub → Settings → Developer settings → New GitHub App. Filling in name, URLs, webhook secret, permissions, events. Generating the private key. Copying App ID & Client ID/Secret. |
| 4  | Install the App on the organization  | 0:40     | Install App → choose org → select repos → Install. Capturing the `installation_id`. |
| 5  | Prepare environment variables        | 1:00     | `base64`-encoding the private key, filling in `.env`, storing secrets with Teller (optional). |
| 6  | Deploy with Docker Compose           | 1:00     | `git clone`, placing `.env`, `docker compose up -d --build`, `docker compose logs -f app`, reverse proxy note. |
| 7  | Connect a code-owner account         | 1:00     | Opening `/app/github/setup?installation_id=...`, completing the OAuth device flow with a CODEOWNER account. |
| 8  | Add `dunebot.yaml` to a repo         | 0:45     | Adding `.github/dunebot.yaml`, committing, showing the Dependabot-style example. |
| 9  | Verify with a test PR                | 0:45     | Opening a Dependabot PR, watching logs, seeing it get approved & merged.      |
| 10 | Outro & links                        | 0:20     | Link to the Installation Guide, `example/dunebot.yml`, Privacy Policy, repo.  |

---

## Recording Checklist

- [ ] 1080p screen capture (GitHub UI + terminal side by side)
- [ ] Test org + test repo prepared so nothing sensitive is shown
- [ ] `.env` values redacted / replaced with placeholders on screen
- [ ] Captions / transcript for accessibility
- [ ] Upload to YouTube (unlisted until reviewed) and paste the link above
- [ ] Replace the placeholder thumbnail with a real one
- [ ] Add the video link to the README's tutorial section

---

## How to Record This Yourself

If you want to produce the video for your team or the community:

1. Follow the [Installation Guide](./installation-guide.md) once on a throwaway
   org so the flow is fresh and you have all the values ready (use placeholders
   on camera).
2. Record each section from the outline above in short takes — they can be
   stitched together in any editor.
3. For the terminal sections, use a tool like
   [asciinema](https://asciinema.org) or
   [terminalizer](https://github.com/faressoft/terminalizer) for clean,
   copy-pasteable recordings.
4. Publish and open a PR updating the placeholder link in this file.