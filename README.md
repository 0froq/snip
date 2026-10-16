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

Snippet cards are small paper strips (default about **240×44** pt, auto width between `[card].min_width` and `[card].max_width`, **1–2 lines** with ellipsis). Drag a card by its surface (`ui.Element.DragWindow()` on the frameless window). Positions are remembered per card for the session (and restored via MyGo `StateKey` when IDs repeat).

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

Settings is a normal window with a title bar, so it can take keyboard focus. The floating rule above floats that window too. To let AeroSpace tile Settings and still float the cards, skip the window titled Settings.

### Settings

Open Settings from the menu bar item, the snip app menu, or the Settings hotkey (default **Cmd+,**). The panel lists the options most people need. Save writes that file and applies the change right away. The non activating panel waits until the next launch. Card chrome stays frameless unless you set `frameless = false` under `[window]` in `config.toml` for advanced debugging.

The file Settings writes is the one snip loaded (`config.toml` in the working directory, or `~/Library/Application Support/snip/config.toml`). If neither file exists yet, Save creates the Application Support copy.

### Permissions

1. **Accessibility** (required for paste simulation): System Settings → Privacy & Security → Accessibility → enable **snip**. On first launch the app logs a reminder; failed paste opens the Accessibility pane.
2. No special permission for clipboard **read** on macOS for this prototype (general pasteboard polling).

### Shortcuts

One shortcut shows or hides every card (`toggle`, default Cmd+Shift+S). Optional direct shortcuts remain for next, previous, paste, delete, flash, and Settings. Turn them off with `direct = false` if they clash with other apps.

A leader shortcut (default Cmd+Shift+Space) starts a short chord mode. Then press one key: `s` toggle, `j` and `k` next and previous, `v` paste, `d` delete, `p` flash, comma settings, `1` to `9` pick a card, Escape cancels. Cards show those hints while the leader is active. It times out (`timeout_ms`).

Double press: under `[leader]`, `double_press_ms` is the longest gap. `double` maps a key to another action, for example `v = "pin"`. The first press waits. A second press in time runs the mapped action.

Flash (`pin`) brings every card forward with the active color, then puts them back.

`dim_overlay` (default off) dims the screen behind the cards while the leader is active. The overlay ignores clicks. When Accessibility can see the focused field, that rectangle stays undimmed.

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
