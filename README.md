# snip (feasibility prototype)

Minimal macOS clipboard manager prototype built with [MyGo](https://mygo.egoist.dev/docs) **native UI** (`ui` package, no webview). Each new plain-text copy spawns a small floating card. Global shortcuts show/hide cards, move selection, paste, and delete.

This repo was built and cross-compiled on Linux; run and exercise it on a Mac.

## Requirements (Mac)

- macOS 12+ (Apple Silicon: `darwin/arm64`)
- Go 1.27+ (MyGo is pinned in `go.mod`; use `go tool mygo` — no separate install)

## Build on Mac (or cross-compile from Linux)

```bash
# From repo root — produces build/darwin-arm64/snip.app
# This default build is ad hoc signed. Leave SNIP_CODESIGN_IDENTITY unset.
go tool mygo build -platform darwin/arm64

# Alternative: single binary (no .app bundle). This ad hoc signature
# does not keep Accessibility across rebuilds.
GOOS=darwin GOARCH=arm64 go build -o snip .
```

### Stable code signature

MyGo signs ad hoc by default (`codesign -s -`). That signature’s hash changes on every rebuild, so macOS TCC treats the new binary as a different app. Accessibility can look enabled while paste still fails.

Create one local code-signing certificate, then reuse it:

1. Open Keychain Access.
2. Certificate Assistant → Create a Certificate…
3. Name: `snip local`
4. Identity Type: Self Signed Root
5. Certificate Type: Code Signing

```bash
# Optional. Unset, this is the same ad hoc build as above.
# Set, the .app is codesigned with that identity after the build.
SNIP_CODESIGN_IDENTITY="snip local" ./scripts/build-mac.sh

# Or re-sign an app you already built:
SNIP_CODESIGN_IDENTITY="snip local" ./scripts/sign-mac.sh
```

An existing Apple Development identity works the same way (`security find-identity -v -p codesigning` lists them). Grant Accessibility once for that signature. Later builds signed with the same identity keep the grant.

`scripts/sign-mac.sh` refuses to ad hoc sign. `SNIP_SIGN_IDENTITY` is accepted as an older name for the same variable. snip asks `AXIsProcessTrusted` without the prompt option first. If paste still cannot run, it opens the Accessibility pane once per launch and only logs after that. A click does not open System Settings again.

## Run on Mac

```bash
open build/darwin-arm64/snip.app
# or: ./snip   (if you used go build)
```

Optional config: copy `config.toml` to `~/Library/Application Support/snip/config.toml` or keep `config.toml` in the working directory.

`max_cards` (default **20**) caps how many snippets are kept. When a new copy would exceed the limit, the **oldest** cards are closed and removed first (FIFO).

Snippet cards are small paper strips in the system sans (`[card].font_family`, default `-apple-system` with PingFang SC) at `[card].font_size` (default 14). A short copy stays on one line and is only as wide as that text plus padding. A long copy grows up to `max_width`, wraps, and ends with an ellipsis after `max_lines`. `min_width` is the narrowest a longer card can be dragged, and it does not stretch a short note. Drag the middle of a card to move it. Drag an edge to change its width or visible line count. Double click a card to paste it. This is on by default and can be turned off.

New cards are placed with `[card].placement` (default **`near-selection`**). snip asks the frontmost app for the focused selection bounds via the Accessibility API and opens the card just below that text (or above when there is no room), clamped to the monitor work area. When that fails (browsers, Electron, terminals), it falls back to near the mouse cursor. **`near-cursor`** skips the selection step. **`fixed`** keeps the old cascade from the top-left corner.

### AeroSpace (tiling window manager)

snip’s bundle ID is **`me.froq.snip`** (`mygo.json` → `identifier`). AeroSpace may tile snip’s floating cards unless you float them. Cards also use MyGo’s **`Panel`** window type when `[window].panel = true` (default), which keeps them out of normal app window lists.

Add to your AeroSpace config ([on-window-detected](https://nikitabobko.github.io/AeroSpace/guide#on-window-detected-callback)):

```toml
on-window-detected = [
    { if = 'test %{app-bundle-id} = me.froq.snip', run = 'layout floating' },
]
```

Legacy table form (still supported by AeroSpace):

```toml
on-window-detected = [
    { if.app-id = 'me.froq.snip', run = 'layout floating' },
]
```

Settings is a focusable frameless paper window. Its header moves the window. The close button, Escape, and Cmd W close it.

### Settings

Open Settings from the menu bar item, the snip app menu, or the Settings hotkey (default **Cmd+,**). The panel lists the options most people need. Save writes that file and applies the change right away. The non activating panel waits until the next launch. Card chrome stays frameless unless you set `frameless = false` under `[window]` in `config.toml` for advanced debugging.

The file Settings writes is the one snip loaded (`config.toml` in the working directory, or `~/Library/Application Support/snip/config.toml`). If neither file exists yet, Save creates the Application Support copy.

### Permissions

1. **Accessibility** (required for paste simulation): System Settings → Privacy & Security → Accessibility → enable **snip**. The app prompts only when `AXIsProcessTrusted` is false. A failed paste opens the Accessibility pane only in that case. Sign with a stable identity (see above) so the grant survives rebuilds.
2. No special permission for clipboard **read** on macOS for this prototype (general pasteboard polling).

### Shortcuts

Each action has a direct global shortcut and a leader key. Either field may be empty. Each one can require a double press. A leader binding can stay active after it runs, which is useful for next and previous.

The leader shortcut is Cmd Shift Space by default. It opens a short chord mode. Each card shows a jump badge centered on its top-right corner (digits 1–9, then 0, then keys that are not already leader actions). The other leader actions are listed in one which-key window. `[leader].which_key_x` and `which_key_y` (0–100) place an anchor on the work area, and `which_key_align_x` / `which_key_align_y` (`start`, `center`, or `end`) choose which point of the panel sits there. The default is the bottom-right corner (`100`, `100`, `end`, `end`), inset by a margin. An old `which_key_corner` value still applies when those four keys are absent. The panel uses a monospace face and a translucent blur. Cards, the which-key window and settings share a 6pt corner radius. Escape cancels the mode. The mode also closes after `timeout_ms`. A sticky leader binding keeps that window up.

Bring to front shows every card and orders it above other windows. It does not flash or recolor them.

### Language and config path

`language` is `en` or `zh` (Settings, menus, and leader hints).

Point snip at a config file with `-config path` or the `SNIP_CONFIG` environment variable. Settings shows the path it loaded.

## What to verify (four capabilities)

### 1. Clipboard watch

1. Start snip.
2. Copy plain text from Safari or Notes (Cmd+C).
3. Expect a new card within ~300 ms showing the text (secrets masked if applicable).
4. Copy again (different text): another card appears.
5. Copy the same text again: the existing card becomes active and pulses. A new card is not created.
6. With `max_cards = 3`, copy four distinct strings: only the three newest cards remain.

### 2. Floating cards

1. Cards should stay above normal windows (`always_on_top`).
2. Click another app: focus stays there; cards remain visible (`ShowInactive`).
3. Switch Spaces: cards follow (`visible_on_all_workspaces`).
4. Toggle `always_on_top` / `visible_on_all_workspaces` in `config.toml`, restart, confirm behavior changes.

### 3. Paste selected card

1. Grant Accessibility.
2. Select a card with Cmd+Shift+] or Cmd+Shift+[, or click the card. The active card uses your active color.
3. Focus a text field in another app.
4. Press Cmd+Shift+V: clipboard gets the card text and Cmd+V is simulated into the frontmost app.

### 4. Screen-capture exclusion + secret masking

1. With `content_protection = true`, start a FaceTime/Zoom/screen recording and confirm cards are **not** visible in the capture (MyGo `SetContentProtection` → `NSWindowSharingNone`).
2. Copy `password=hunter2` or a `sk-…` OpenAI-style key; the card should show masked text, while paste still uses the real value.
3. Copy from a password manager that sets `org.nspasteboard.ConcealedType`: the card masks the text even when `content_protection` is off. Transient and auto generated markers can skip creating a card (see `[pasteboard]` in Settings or `config.toml`).

## Tests (pure logic, any OS)

```bash
go test ./...
```

## Project layout

| Path | Role |
|------|------|
| `main.go` | MyGo app entry |
| `app/app.go` | Cards, shortcuts, clipboard loop, settings window |
| `internal/macpaste/` | Darwin pasteboard changeCount + CGEvent Cmd+V (purego) |
| `internal/macplace/` | Card placement (AX selection bounds, cursor, screen clamp) |
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
