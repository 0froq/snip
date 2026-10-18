package macplace

// WindowCornerRadius is the corner radius of every snip window, in points.
// It is smaller than the macOS window radius so cards, the which-key panel
// and settings share one edge.
const WindowCornerRadius = 6

// CornerCurveContinuous is CALayer's kCACornerCurveContinuous ("continuous"):
// the squircle AppKit uses for cornerCurve = .continuous. mygo already fills
// with that curve on macOS; the window mask has to use the same one or it
// clips the fill back to a quarter circle.
const CornerCurveContinuous = "continuous"
