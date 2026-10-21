#!/bin/sh
# Sign snip with a stable codesigning identity so macOS TCC keeps the
# Accessibility grant across rebuilds. Ad hoc signatures (codesign -s -)
# change every build and look like a new app.
set -eu

APP="${1:-build/darwin-arm64/snip.app}"
# SNIP_CODESIGN_IDENTITY is the optional stable identity. SNIP_SIGN_IDENTITY
# is the older name for the same thing.
IDENTITY="${SNIP_CODESIGN_IDENTITY:-${SNIP_SIGN_IDENTITY:-}}"

if [ ! -d "$APP" ]; then
	echo "app not found: $APP" >&2
	echo "build first: go tool mygo build -platform darwin/arm64 -sign \"snip local\"" >&2
	exit 1
fi

if [ -z "$IDENTITY" ]; then
	IDENTITY=$(security find-identity -v -p codesigning | sed -n 's/.*"\(snip local\)"/\1/p' | head -n 1)
fi

if [ -z "$IDENTITY" ]; then
	echo "No stable signing identity named \"snip local\"." >&2
	echo "Create one once, then re-run this script:" >&2
	echo "  1. Open Keychain Access" >&2
	echo "  2. Certificate Assistant → Create a Certificate…" >&2
	echo "  3. Name: snip local" >&2
	echo "  4. Identity Type: Self Signed Root" >&2
	echo "  5. Certificate Type: Code Signing" >&2
	echo "Or pass an existing identity: SNIP_CODESIGN_IDENTITY='Apple Development: Name' $0" >&2
	exit 1
fi

codesign --force --sign "$IDENTITY" --identifier "me.froq.snip" "$APP"
echo "Signed $APP with $IDENTITY"
echo "Grant Accessibility once for this signature. Later rebuilds signed the same way keep it."
