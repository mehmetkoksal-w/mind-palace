package dashboard

import (
	"net/http"
	"time"
)

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
	})
}

// faviconSVG is the Mind Palace logo optimized for favicon display.
// Based on the official logo from assets/logo/logo.svg.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
<g fill="#6B5B95">
<path d="M499.51,335.772l-46.048-130.234C445.439,90.702,349.802,0,232.917,0C110.768,0,11.759,99.019,11.759,221.158v69.684C11.759,412.982,110.768,512,232.917,512c100.571,0,185.406-67.154,212.256-159.054h42.186c4.181,0,8.104-2.032,10.518-5.45C500.291,344.088,500.895,339.712,499.51,335.772z M328.82,214.59c2.511,14.166-2.495,37.128-47.051,37.128c-21.355,0-51.382,0-61.731,0c0,33.737-50.903,25.819-68.779,25.819c-17.911,0-20.832-19.19-17.565-24.178c-55.846-0.417-63.701-58.749-49.988-84.196C89.573,99.96,159.585,50.77,229.242,50.77c91.661,0,85.59,30.009,95.805,30.861c25.03,2.023,59.219,31.269,59.219,65.415C384.267,181.244,370.074,214.59,328.82,214.59z"/>
</g>
<g fill="#FFFFFF" opacity="0.95">
<circle cx="180" cy="200" r="20"/>
<circle cx="280" cy="180" r="15"/>
<circle cx="230" cy="130" r="12"/>
</g>
</svg>`

// handleFavicon serves a simple SVG favicon.
func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(faviconSVG))
}
