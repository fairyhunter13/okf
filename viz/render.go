package viz

import (
	_ "embed"
	"html"
	"strings"
)

// Embedded rather than fetched. These three files are the whole viewer, and
// substitution beats text/template here because the JS and CSS are full of the
// braces a template would try to read.
var (
	//go:embed assets/viz.html
	tmpl string
	//go:embed assets/viz.css
	css string
	//go:embed assets/viz.js
	js string
)

// render fills the page. The data goes into a <script type="application/json">
// block, where the only sequence that could break out is "</script>" — and
// encoding/json escapes "<" to < before it can appear.
func render(name, data string) string {
	page := strings.ReplaceAll(tmpl, "__NAME__", html.EscapeString(name))
	page = strings.Replace(page, "/*__CSS__*/", css, 1)
	page = strings.Replace(page, "/*__DATA__*/", data, 1)
	return strings.Replace(page, "/*__JS__*/", js, 1)
}
