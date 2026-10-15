# snip (feasibility prototype)

Minimal macOS clipboard manager prototype built with [MyGo](https://mygo.egoist.dev/docs) **native UI** (`ui` package, no webview). Each new plain-text copy spawns a small floating card. Global shortcuts show/hide cards, move selection, paste, and delete.

This repo was built and cross-compiled on Linux; run and exercise it on a Mac.

## Requirements (Mac)

- macOS 12+ (Apple Silicon: `darwin/arm64`)
- Go 1.27+ (MyGo is pinned in `go.mod`; use `go tool mygo` — no separate install)

## Build on Mac (or cross-compile from Linux)

```bash
# From repo root — produces build/darwin-arm64/snip.app
go tool mygo build -platform darwin/arm64

# Alternative: single binary (no .app bundle)
GOOS=darwin GOARCH=arm64 go build -o snip .
```

## Run on Mac

```bash
open build/darwin-arm64/snip.app
# or: ./snip   (if you used go build)
```

Optional config: copy `config.toml` to `~/Library/Application Support/snip/config.toml` or keep `config.toml` in the working directory.

`max_cards` (default **20**) caps how many snippets are kept. When a new copy would exceed the limit, the **oldest** cards are closed and removed first (FIFO).

### Permissions

1. **Accessibility** (required for paste simulation): System Settings → Privacy & Security → Accessibility → enable **snip**. On first launch the app logs a reminder; failed paste opens the Accessibility pane.
2. No special permission for clipboard **read** on macOS for this prototype (general pasteboard polling).

### Global shortcuts (defaults)

| Action | Shortcut |
|--------|----------|
| Show all cards | Cmd+Shift+S |
| Hide all cards | Cmd+Shift+H |
| Select next / previous | Cmd+Shift+] / Cmd+Shift+[ |
| Paste selected card | Cmd+Shift+V |
| Delete selected card | Cmd+Shift+Backspace |

Change these under `[shortcuts]` in `config.toml`. If registration fails, MyGo logs an error (shortcut taken by another app).

## What to verify (four capabilities)

### 1. Clipboard watch

1. Start snip.
2. Copy plain text from Safari or Notes (Cmd+C).
3. Expect a new card within ~300 ms showing the text (secrets masked if applicable).
4. Copy again (different text): another card appears.
5. Copy the **same** text twice: two cards (changeCount-based detection).
6. With `max_cards = 3`, copy four distinct strings: only the three newest cards remain.

### 2. Floating cards

1. Cards should stay above normal windows (`always_on_top`).
2. Click another app: focus stays there; cards remain visible (`ShowInactive`).
3. Switch Spaces: cards follow (`visible_on_all_workspaces`).
4. Toggle `always_on_top` / `visible_on_all_workspaces` in `config.toml`, restart, confirm behavior changes.

### 3. Paste selected card

1. Grant Accessibility.
2. Select a card (Cmd+Shift+] / [ — accent border).
3. Focus a text field in another app.
4. Press Cmd+Shift+V: clipboard gets the card text and Cmd+V is simulated into the frontmost app.

### 4. Screen-capture exclusion + secret masking

1. With `content_protection = true`, start a FaceTime/Zoom/screen recording and confirm cards are **not** visible in the capture (MyGo `SetContentProtection` → `NSWindowSharingNone`).
2. Copy `password=hunter2` or a `sk-…` OpenAI-style key; the card should show masked text, while paste still uses the real value.

## Tests (pure logic, any OS)

```bash
go test ./...
```

## Project layout

| Path | Role |
|------|------|
| `main.go` | MyGo app entry |
| `app/app.go` | Cards, shortcuts, clipboard loop |
| `internal/macpaste/` | Darwin pasteboard changeCount + CGEvent Cmd+V (purego) |
| `secrets/` | Regex / entropy masking |
| `store/` | In-memory card store |
| `config/` | TOML config |

## Capability matrix (prototype conclusion)

| # | Feature | Status | Evidence |
|---|---------|--------|----------|
| 1 | Clipboard watch (changeCount ~300ms) | **Bridged** | `internal/macpaste/darwin.go` — `ChangeCount`, `ReadPlainText`; MyGo exposes `Clipboard.ReadText` only ([data-transfer docs](https://mygo.egoist.dev/docs/data-transfer)), no watch API |
| 2 | Floating non-activating cards, all Spaces | **Native** | `app/app.go` — `WindowOptions`, `ShowInactive`, `SetVisibleOnAllWorkspaces`, `SetActivationPolicy(Accessory)` |
| 3 | Paste + global shortcuts | **Mixed** | Shortcuts: `mygo.GlobalShortcut` (**native**). Paste: `mygo.Clipboard.WriteText` (**native**) + `macpaste.PostCommandV` (**bridged**, CGEvent via purego) |
| 4 | Capture exclusion + masking | **Mixed** | `SetContentProtection` (**native**, maps to `setSharingType:` in MyGo `internal/darwin/window.go`). Masking: `secrets/` (**app logic**) |

## Open risks

- **Accessibility**: paste simulation fails without trust; macOS may still block events in some secure fields.
- **Shortcut conflicts**: global accelerators can collide with other tools; no UI to rebind yet (edit TOML).
- **Clipboard**: only plain UTF-8 text; images/files ignored. Polling uses CPU every 300 ms.
- **Many cards**: one `Window` per snippet; FIFO eviction when `max_cards` is exceeded.
- **Panel vs window**: cards use regular always-on-top windows, not `Panel` (launcher-style); behavior on full-screen apps should be validated on hardware.
- **Linux build**: `snip.app` is macOS-only; running `go tool mygo dev` on Linux will not exercise clipboard or paste bridges.
