package config

// ResolveFont is the face a card measures and the text view draws.
// A leading generic (-apple-system, system-ui, sans-serif, serif,
// monospace) is the system font: names after it only cover glyphs that
// face lacks. A leading concrete family is that face. system is true when
// the text view should use the system font.
func ResolveFont(family string) (name string, system bool) {
	start := 0
	for i := 0; i <= len(family); i++ {
		if i != len(family) && family[i] != ',' {
			continue
		}
		part := trimFamily(family[start:i])
		start = i + 1
		if part == "" {
			continue
		}
		if genericFamily(part) {
			return "", true
		}
		return part, false
	}
	return "", true
}

func genericFamily(name string) bool {
	switch lowerFamily(name) {
	case "-apple-system", "system-ui", "sans-serif", "serif", "monospace":
		return true
	default:
		return false
	}
}

func lowerFamily(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func trimFamily(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
