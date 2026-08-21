package viz

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// The subset OKF concepts are written in. A general markdown library would be
// a dependency, a supply chain and a CVE feed for a page that renders eleven
// constructs; anything outside this set falls through as escaped text, which
// reads as the source rather than as damage.
var (
	mdLink   = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)\)`)
	mdBold   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalic = regexp.MustCompile(`(^|\W)_([^_]+)_(\W|$)`)
	mdCode   = regexp.MustCompile("`([^`]+)`")
	mdOrder  = regexp.MustCompile(`^\d+\.\s+`)
)

// Markdown renders a concept body to HTML. Every text run is escaped before a
// tag is introduced, so a body containing markup renders it rather than running
// it.
func Markdown(src string) string {
	var b strings.Builder
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "```"):
			i = fence(&b, lines, i)
		case strings.HasPrefix(line, "#"):
			heading(&b, line)
		case isRule(line):
			b.WriteString("<hr>")
		case strings.HasPrefix(strings.TrimSpace(line), "|"):
			i = table(&b, lines, i)
		case isBullet(line) || mdOrder.MatchString(strings.TrimSpace(line)):
			i = list(&b, lines, i)
		case strings.TrimSpace(line) == "":
		default:
			i = paragraph(&b, lines, i)
		}
	}
	return b.String()
}

func fence(b *strings.Builder, lines []string, i int) int {
	lang := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "```"))
	var body []string
	j := i + 1
	for ; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "```"); j++ {
		body = append(body, lines[j])
	}
	fmt.Fprintf(b, "<pre class=\"code\" data-lang=%q><code>%s</code></pre>",
		lang, html.EscapeString(strings.Join(body, "\n")))
	return j
}

func heading(b *strings.Builder, line string) {
	n := len(line) - len(strings.TrimLeft(line, "#"))
	if n > 6 {
		n = 6
	}
	// Headings shift down one: the concept title is the page's h1, so a body
	// `#` is a section inside it, not a second document title.
	fmt.Fprintf(b, "<h%d>%s</h%d>", min(n+1, 6), inline(strings.TrimSpace(line[n:])), min(n+1, 6))
}

func table(b *strings.Builder, lines []string, i int) int {
	var rows [][]string
	j := i
	for ; j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|"); j++ {
		rows = append(rows, cells(lines[j]))
	}
	b.WriteString("<div class=\"tw\"><table>")
	for r, row := range rows {
		if r == 1 && isDivider(row) {
			continue
		}
		tag := "td"
		if r == 0 {
			tag = "th"
		}
		b.WriteString("<tr>")
		for _, c := range row {
			fmt.Fprintf(b, "<%s>%s</%s>", tag, inline(c), tag)
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</table></div>")
	return j - 1
}

func list(b *strings.Builder, lines []string, i int) int {
	tag := "ul"
	if mdOrder.MatchString(strings.TrimSpace(lines[i])) {
		tag = "ol"
	}
	fmt.Fprintf(b, "<%s>", tag)
	j := i
	for ; j < len(lines); j++ {
		t := strings.TrimSpace(lines[j])
		switch {
		case isBullet(lines[j]):
			t = strings.TrimSpace(t[1:])
		case mdOrder.MatchString(t):
			t = mdOrder.ReplaceAllString(t, "")
		default:
			if t == "" {
				continue
			}
			goto done
		}
		fmt.Fprintf(b, "<li>%s</li>", inline(t))
	}
done:
	fmt.Fprintf(b, "</%s>", tag)
	return j - 1
}

func paragraph(b *strings.Builder, lines []string, i int) int {
	var run []string
	j := i
	for ; j < len(lines) && !isBreak(lines[j]); j++ {
		run = append(run, strings.TrimSpace(lines[j]))
	}
	fmt.Fprintf(b, "<p>%s</p>", inline(strings.Join(run, " ")))
	return j - 1
}

// inline escapes first and marks up second, so `<b>` in a body is text.
// Code spans are lifted out before the emphasis passes, because a span holding
// `**` is an example of the syntax and not a use of it.
func inline(s string) string {
	var spans []string
	s = mdCode.ReplaceAllStringFunc(s, func(m string) string {
		spans = append(spans, html.EscapeString(mdCode.FindStringSubmatch(m)[1]))
		return fmt.Sprintf("\x00%d\x00", len(spans)-1)
	})
	s = html.EscapeString(s)
	s = mdLink.ReplaceAllString(s, `<a href="$2">$1</a>`)
	s = mdBold.ReplaceAllString(s, "<strong>$1</strong>")
	s = mdItalic.ReplaceAllString(s, "$1<em>$2</em>$3")
	for i, c := range spans {
		s = strings.ReplaceAll(s, fmt.Sprintf("\x00%d\x00", i), "<code>"+c+"</code>")
	}
	return s
}

func cells(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	out := strings.Split(t, "|")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func isDivider(row []string) bool {
	for _, c := range row {
		if strings.Trim(c, ":- ") != "" {
			return false
		}
	}
	return len(row) > 0
}

func isBullet(line string) bool {
	t := strings.TrimSpace(line)
	return len(t) > 1 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && t[1] == ' '
}

func isRule(line string) bool {
	t := strings.TrimSpace(line)
	return len(t) >= 3 && strings.Trim(t, "-") == "" || strings.Trim(t, "*") == "" && len(t) >= 3
}

func isBreak(line string) bool {
	t := strings.TrimSpace(line)
	return t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") ||
		strings.HasPrefix(t, "```") || isBullet(line) || mdOrder.MatchString(t)
}
