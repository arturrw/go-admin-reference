package media

import (
	"fmt"
	"strings"
)

// Glyphs are 24×24 line icons (adapted from Lucide, ISC licence) used as the
// subject of the generated product shots.
var glyphs = map[string]string{
	"Audio":       `<path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 18 0v7a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3"/>`,
	"Wearables":   `<circle cx="12" cy="12" r="6"/><path d="M12 10v2l1 1"/><path d="m16.13 7.66-.81-4.05a2 2 0 0 0-2-1.61h-2.68a2 2 0 0 0-2 1.61l-.78 4.05"/><path d="m7.88 16.36.8 4a2 2 0 0 0 2 1.61h2.72a2 2 0 0 0 2-1.61l.81-4.05"/>`,
	"Lighting":    `<path d="m14 5-3 3 2 7 8-8-7-2Z"/><path d="m14 5-3 3-3-3 3-3 3 3Z"/><path d="M9.5 6.5 4 12l3 6"/><path d="M3 22v-2c0-1.1.9-2 2-2h4a2 2 0 0 1 2 2v2H3Z"/>`,
	"Home":        `<path d="M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8"/><path d="M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`,
	"Computing":   `<path d="M20 16V7a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v9m16 0H4m16 0 1.28 2.55a1 1 0 0 1-.9 1.45H3.62a1 1 0 0 1-.9-1.45L4 16"/>`,
	"Accessories": `<path d="M4 10a4 4 0 0 1 4-4h8a4 4 0 0 1 4 4v10a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2Z"/><path d="M8 10h8"/><path d="M8 18h8"/><path d="M8 22v-6a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v6"/><path d="M9 6V4a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2"/>`,
}

type shot struct {
	scale, rotate, dx, dy float64
	hueShift              int
	label                 string
}

// Three "camera angles" per product.
var shots = []shot{
	{scale: 17, rotate: 0, dx: 0, dy: -10, hueShift: 0, label: "front"},
	{scale: 15, rotate: -14, dx: -30, dy: 0, hueShift: 28, label: "angle"},
	{scale: 25, rotate: 8, dx: 40, dy: 30, hueShift: -22, label: "detail"},
}

func ShotCount() int { return len(shots) }

func ShotLabel(variant int) string { return shots[variant%len(shots)].label }

// ProductSVG renders an 800×800 studio-style illustration for a product.
func ProductSVG(category string, hue, variant int) []byte {
	g, ok := glyphs[category]
	if !ok {
		g = glyphs["Accessories"]
	}
	s := shots[((variant%len(shots))+len(shots))%len(shots)]
	h := hue + s.hueShift
	cx, cy := 400+s.dx, 380+s.dy

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 800" width="800" height="800">
<defs>
  <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
    <stop offset="0" stop-color="hsl(%[1]d 42%% 17%%)"/><stop offset="1" stop-color="hsl(%[1]d 38%% 6%%)"/>
  </linearGradient>
  <radialGradient id="glow" cx="%.0[2]f" cy="%.0[3]f" r="360" gradientUnits="userSpaceOnUse">
    <stop offset="0" stop-color="hsl(%[1]d 85%% 60%%)" stop-opacity=".45"/><stop offset="1" stop-color="hsl(%[1]d 85%% 60%%)" stop-opacity="0"/>
  </radialGradient>
  <linearGradient id="stroke" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="hsl(%[1]d 95%% 88%%)"/><stop offset="1" stop-color="hsl(%[1]d 80%% 62%%)"/>
  </linearGradient>
  <radialGradient id="floor" cx="400" cy="660" r="260" gradientUnits="userSpaceOnUse" gradientTransform="matrix(1 0 0 .18 0 541)">
    <stop offset="0" stop-color="#000" stop-opacity=".55"/><stop offset="1" stop-color="#000" stop-opacity="0"/>
  </radialGradient>
  <pattern id="grid" width="32" height="32" patternUnits="userSpaceOnUse"><path d="M32 0H0v32" fill="none" stroke="#fff" stroke-opacity=".035"/></pattern>
  <filter id="blur" x="-50%%" y="-50%%" width="200%%" height="200%%"><feGaussianBlur stdDeviation=".8"/></filter>
</defs>
<rect width="800" height="800" fill="url(#bg)"/>
<rect width="800" height="800" fill="url(#grid)"/>
<rect width="800" height="800" fill="url(#glow)"/>
<ellipse cx="400" cy="660" rx="260" ry="46" fill="url(#floor)"/>
`, h, cx, cy)
	transform := fmt.Sprintf("translate(%.0f %.0f) rotate(%.0f) scale(%.1f) translate(-12 -12)", cx, cy, s.rotate, s.scale)
	fmt.Fprintf(&b, `<g transform="%s" fill="none" stroke-linecap="round" stroke-linejoin="round">
  <g stroke="hsl(%d 90%% 60%%)" stroke-width="2.4" opacity=".55" filter="url(#blur)">%s</g>
  <g stroke="url(#stroke)" stroke-width="1.15" fill="hsl(%d 60%% 50%% / .08)">%s</g>
</g>
</svg>`, transform, h, g, h, g)
	return []byte(b.String())
}
