package secrets

import (
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Config drives masking behavior.
type Config struct {
	Enabled         bool
	ExtraPatterns   []string
	MinEntropyBits  float64
	MinSecretLength int
}

var builtinPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:api[_-]?key|apikey|secret|token|password|passwd|bearer)\b[\s:='"]+[\w\-./+=]{8,}`),
	regexp.MustCompile(`\bsk-[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bghp_[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bgho_[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bxox[baprs]-[a-zA-Z0-9\-]{10,}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\b`),
}

// DisplayText returns text safe to show on a card.
func DisplayText(raw string, cfg Config) string {
	if !cfg.Enabled || raw == "" {
		return raw
	}
	out := raw
	for _, re := range builtinPatterns {
		out = re.ReplaceAllStringFunc(out, maskMatch)
	}
	for _, pat := range cfg.ExtraPatterns {
		re, err := regexp.Compile(pat)
		if err != nil {
			continue
		}
		out = re.ReplaceAllStringFunc(out, maskMatch)
	}
	if cfg.MinSecretLength > 0 && cfg.MinEntropyBits > 0 {
		out = maskHighEntropyTokens(out, cfg.MinSecretLength, cfg.MinEntropyBits)
	}
	return out
}

func maskMatch(s string) string {
	if len(s) <= 4 {
		return "••••"
	}
	return s[:2] + strings.Repeat("•", len(s)-4) + s[len(s)-2:]
}

func maskHighEntropyTokens(text string, minLen int, minBits float64) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if isTokenChar(r) {
			start := i
			i += size
			for i < len(text) {
				r2, sz := utf8.DecodeRuneInString(text[i:])
				if !isTokenChar(r2) {
					break
				}
				i += sz
			}
			tok := text[start:i]
			if len(tok) >= minLen && shannonEntropy(tok) >= minBits {
				b.WriteString(maskMatch(tok))
			} else {
				b.WriteString(tok)
			}
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func isTokenChar(r rune) bool {
	return r == '_' || r == '-' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := make(map[rune]int)
	for _, r := range s {
		freq[r]++
	}
	var ent float64
	n := float64(len(s))
	for _, c := range freq {
		p := float64(c) / n
		ent -= p * math.Log2(p)
	}
	return ent
}
