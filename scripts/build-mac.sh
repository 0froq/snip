#!/bin/sh
# Build the macOS app bundle. The default path is unchanged: MyGo's ad hoc
# signature is left as-is. When SNIP_CODESIGN_IDENTITY is set, the .app is
# re-signed with that identity so the Accessibility grant survives rebuilds.
set -eu
cd "$(dirname "$0")/.."
go tool mygo build -platform darwin/arm64
if [ -n "${SNIP_CODESIGN_IDENTITY:-}" ]; then
	./scripts/sign-mac.sh build/darwin-arm64/snip.app
fi
