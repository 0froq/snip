package config

// EffectiveFrameless reports whether snippet cards are frameless (default true).
func (w Window) EffectiveFrameless() bool {
	if w.Frameless == nil {
		return true
	}
	return *w.Frameless
}

// EffectivePanel reports whether snippet cards use a non activating panel (default true).
func (w Window) EffectivePanel() bool {
	if w.Panel == nil {
		return true
	}
	return *w.Panel
}

func boolPtr(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}
