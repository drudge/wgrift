//go:build js && wasm

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// setPageTitle updates the browser tab title.
func setPageTitle(title string) {
	if title == "" {
		js.Global().Get("document").Set("title", "wgRift")
	} else {
		js.Global().Get("document").Set("title", title+" · wgRift")
	}
}

// Card wraps content in an elevated surface.
func Card(children ...loom.Node) loom.Node {
	return Div(
		append([]loom.Node{
			Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg p-6"}),
		}, children...)...,
	)
}

// CardHeader is a card title row.
func CardHeader(title string, extra ...loom.Node) loom.Node {
	children := []loom.Node{
		Apply(Attr{"class": "flex flex-wrap items-center justify-between gap-3 mb-5"}),
		H3(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text(title)),
	}
	children = append(children, extra...)
	return Div(children...)
}

// PageHeader renders a bold page title with optional actions.
func PageHeader(title, subtitle string, actions ...loom.Node) loom.Node {
	children := []loom.Node{
		Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-8"}),
		Div(
			H2(Apply(Attr{"class": "text-2xl font-bold text-ink-1 tracking-tight"}), Text(title)),
			func() loom.Node {
				if subtitle != "" {
					return P(Apply(Attr{"class": "text-sm text-ink-3 mt-1"}), Text(subtitle))
				}
				return Span()
			}(),
		),
	}
	if len(actions) > 0 {
		children = append(children, Div(
			Apply(Attr{"class": "flex items-center gap-2"}),
			Fragment(actions...),
		))
	}
	return Div(children...)
}

// Btn renders a button with variant styling.
func Btn(label string, variant string, handler func()) loom.Node {
	class := "inline-flex items-center justify-center font-medium rounded-md transition-all duration-100 cursor-pointer "
	switch variant {
	case "primary":
		class += "text-xs px-4 py-2 border border-wg-500/40 text-wg-400 bg-wg-600/10 hover:bg-wg-600/20 hover:border-wg-500/60 active:bg-wg-600/25"
	case "danger":
		class += "text-xs px-3.5 py-2 text-red-400/80 hover:text-red-400 hover:bg-red-500/10"
	case "ghost":
		class += "text-xs px-3.5 py-2 text-ink-2 hover:text-ink-1 hover:bg-surface-2"
	default:
		class += "text-xs px-3.5 py-2 text-ink-2 hover:text-ink-1 hover:bg-surface-2"
	}

	return Button(
		Apply(Attr{"class": class}),
		Apply(On{"click": func() { handler() }}),
		Text(label),
	)
}

// IconBtn renders a small icon-only button with tooltip.
func IconBtn(iconName string, title string, handler func()) loom.Node {
	return Button(
		Apply(Attr{
			"class": "w-8 h-8 rounded-md flex items-center justify-center text-ink-4 hover:text-ink-1 hover:bg-surface-2 transition-all duration-100",
			"title": title,
		}),
		Apply(On{"click": func() { handler() }}),
		Icon(iconName, 15),
	)
}

// IconBtnDanger renders a small icon-only danger button with tooltip.
func IconBtnDanger(iconName string, title string, handler func()) loom.Node {
	return Button(
		Apply(Attr{
			"class": "w-8 h-8 rounded-md flex items-center justify-center text-ink-4 hover:text-red-400 hover:bg-red-500/10 transition-all duration-100",
			"title": title,
		}),
		Apply(On{"click": func() { handler() }}),
		Icon(iconName, 15),
	)
}

// StatusDot renders a colored status indicator.
func StatusDot(enabled, connected bool) loom.Node {
	if !enabled {
		return Span(Apply(Attr{"class": "inline-flex items-center justify-center w-2.5 h-2.5 rounded-full border-[1.5px] border-red-400/50 bg-transparent"}),
			Span(Apply(Attr{"class": "block w-[6px] h-[1.5px] bg-red-400/50 rounded-full -rotate-45"})),
		)
	}
	if connected {
		return Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-green-500 status-pulse"}))
	}
	return Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-ink-4"}))
}

// Badge renders a small label.
func Badge(label string, color string) loom.Node {
	class := "inline-block px-2 py-0.5 text-[10px] font-semibold rounded "
	switch color {
	case "teal":
		class += "bg-wg-600/15 text-wg-400"
	case "amber":
		class += "bg-amber-500/10 text-amber-400"
	case "red":
		class += "bg-red-500/10 text-red-400"
	case "emerald":
		class += "bg-green-500/10 text-green-400"
	default:
		class += "bg-surface-3 text-ink-3"
	}
	return Span(Apply(Attr{"class": class}), Text(label))
}

// Tooltip wraps a trigger node with a hover popover listing items.
func Tooltip(trigger loom.Node, items []string) loom.Node {
	rows := make([]loom.Node, len(items))
	for i, item := range items {
		rows[i] = Div(
			Apply(Attr{"class": "py-[2px] text-ink-2"}),
			Text(item),
		)
	}
	return Span(
		Apply(Attr{"class": "tooltip-trigger relative cursor-pointer", "tabindex": "0"}),
		trigger,
		Div(
			Apply(Attr{"class": "tooltip-body absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 z-50 pointer-events-none"}),
			Div(
				Apply(Attr{"class": "bg-surface-3 border border-line-3 rounded-md px-2.5 py-1.5 text-[11px] leading-none font-mono whitespace-nowrap"}),
				Fragment(rows...),
			),
		),
	)
}

// MonoText renders text in monospace.
func MonoText(content string) loom.Node {
	return Span(Apply(Attr{"class": "font-mono text-sm"}), Text(content))
}

// FormField renders a label + input.
func FormField(label, inputType, placeholder string, value Accessor[string], onInput func(string)) loom.Node {
	initVal := value()
	attrs := Attr{
		"class":       "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/40 focus:ring-1 focus:ring-wg-600/15 transition-colors font-mono",
		"type":        inputType,
		"placeholder": placeholder,
	}
	if initVal != "" {
		attrs["value"] = initVal
	}
	return Div(
		Apply(Attr{"class": "mb-4"}),
		Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text(label)),
		Input(
			Apply(attrs),
			Apply(On{"input": func(evt *EventInput) {
				onInput(evt.InputValue())
			}}),
		),
	)
}

// FocusInput focuses the first input matching the CSS selector after render.
func FocusInput(selector string) {
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		if el := js.Global().Get("document").Call("querySelector", selector); el.Truthy() {
			el.Call("focus")
		}
		return nil
	}), 100)
}

// FormFieldWithHelp renders a label + input + help text.
func FormFieldWithHelp(label, inputType, placeholder, helpText string, value Accessor[string], onInput func(string)) loom.Node {
	initVal := value()
	attrs := Attr{
		"class":       "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/40 focus:ring-1 focus:ring-wg-600/15 transition-colors font-mono",
		"type":        inputType,
		"placeholder": placeholder,
	}
	if initVal != "" {
		attrs["value"] = initVal
	}
	helpClass := "text-xs text-ink-4 mt-1.5"
	if helpText == "" {
		helpClass = "hidden"
	}
	return Div(
		Apply(Attr{"class": "mb-4"}),
		Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text(label)),
		Input(
			Apply(attrs),
			Apply(On{"input": func(evt *EventInput) {
				onInput(evt.InputValue())
			}}),
		),
		P(Apply(Attr{"class": helpClass}), Text(helpText)),
	)
}

// normalizeCIDR attempts to parse a CIDR or bare IP and return a normalized CIDR string.
// defaultPrefix is appended to bare IPs (e.g., "/24" for interfaces, "/32" for peers).
func normalizeCIDR(input, defaultPrefix string) (string, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return s, true
	}
	if _, _, err := net.ParseCIDR(s); err == nil {
		return s, true
	}
	if ip := net.ParseIP(s); ip != nil {
		cidr := s + defaultPrefix
		if _, _, err := net.ParseCIDR(cidr); err == nil {
			return cidr, true
		}
	}
	return s, false
}

// splitCIDR splits "10.100.0.1/24" into ("10.100.0.1", "24").
// If no prefix is present, returns the defaultPrefix (without leading slash).
func splitCIDR(val, defaultPrefix string) (string, string) {
	val = strings.TrimSpace(val)
	dflt := strings.TrimPrefix(defaultPrefix, "/")
	if val == "" {
		return "", dflt
	}
	if parts := strings.SplitN(val, "/", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}
	return val, dflt
}

// cidrSummary computes a human-readable summary for a CIDR (e.g. "10.100.0.0 – 10.100.0.255 · 254 hosts").
func cidrSummary(ipStr, prefixStr string) string {
	if ipStr == "" || prefixStr == "" {
		return ""
	}
	prefix, err := strconv.Atoi(prefixStr)
	if err != nil || prefix < 0 || prefix > 32 {
		return ""
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	if prefix == 32 {
		return "Single host"
	}
	mask := net.CIDRMask(prefix, 32)
	network := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		network[i] = ip4[i] & mask[i]
	}
	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | ^mask[i]
	}
	hosts := (1 << (32 - prefix)) - 2
	if hosts < 1 {
		hosts = 1
	}
	return fmt.Sprintf("%s – %s · %d %s", network, broadcast, hosts, pluralize(hosts, "host", "hosts"))
}

// cidrPrefixOptions defines the common CIDR prefix lengths shown in the dropdown.
var cidrPrefixOptions = []struct {
	Value string
	Label string
}{
	{"8", "/8"},
	{"12", "/12"},
	{"16", "/16"},
	{"20", "/20"},
	{"24", "/24"},
	{"25", "/25"},
	{"26", "/26"},
	{"27", "/27"},
	{"28", "/28"},
	{"29", "/29"},
	{"30", "/30"},
	{"31", "/31"},
	{"32", "/32"},
}

// prefixHostHint returns a short host count string for a CIDR prefix (e.g. "254 hosts").
func prefixHostHint(pfx string) string {
	n, err := strconv.Atoi(pfx)
	if err != nil || n < 0 || n > 32 {
		return "—"
	}
	if n == 32 {
		return "Single host"
	}
	if n == 31 {
		return "2 hosts (point-to-point)"
	}
	hosts := (1 << (32 - n)) - 2
	if hosts >= 1000000 {
		return fmt.Sprintf("~%dM usable hosts", hosts/1000000)
	}
	if hosts >= 1000 {
		return fmt.Sprintf("~%dK usable hosts", hosts/1000)
	}
	return fmt.Sprintf("%d usable hosts", hosts)
}

var cidrFieldSeq int

// CIDRField renders a compound IP + prefix input with inline validation.
// For defaultPrefix="/32", it shows a static /32 suffix (host-only mode).
// For other prefixes, it shows a dropdown of common prefix lengths with info tooltip and subnet summary.
func CIDRField(label, placeholder, helpText, defaultPrefix string, value Accessor[string], onInput func(string)) loom.Node {
	cidrFieldSeq++
	fieldID := fmt.Sprintf("cidr-field-%d", cidrFieldSeq)
	hostOnly := defaultPrefix == "/32"

	initVal := value()
	initIP, initPrefix := splitCIDR(initVal, defaultPrefix)
	ipPart, setIPPart := Signal(initIP)
	prefixPart, setPrefixPart := Signal(initPrefix)
	errHint, setErrHint := Signal("")

	var summaryText func() string
	var setSummaryText func(string)
	if !hostOnly {
		summaryText, setSummaryText = Signal(cidrSummary(initIP, initPrefix))
	}

	placeholderIP, _ := splitCIDR(placeholder, defaultPrefix)

	syncValue := func() {
		ip := ipPart()
		pfx := prefixPart()
		if ip != "" && pfx != "" {
			onInput(ip + "/" + pfx)
		} else if ip != "" {
			onInput(ip)
		} else {
			onInput("")
		}
		if !hostOnly {
			setSummaryText(cidrSummary(ip, pfx))
		}
	}

	setContainerError := func(hasErr bool) {
		container := js.Global().Get("document").Call("getElementById", fieldID)
		if !container.Truthy() {
			return
		}
		cl := container.Get("classList")
		if hasErr {
			cl.Call("remove", "border-line-1", "focus-within:border-wg-600/40", "focus-within:ring-wg-600/15")
			cl.Call("add", "border-red-500/40", "focus-within:border-red-500/40", "focus-within:ring-red-500/15")
		} else {
			cl.Call("remove", "border-red-500/40", "focus-within:border-red-500/40", "focus-within:ring-red-500/15")
			cl.Call("add", "border-line-1", "focus-within:border-wg-600/40", "focus-within:ring-wg-600/15")
		}
	}

	clearError := func() {
		if errHint() != "" {
			setErrHint("")
			setContainerError(false)
		}
	}

	validateIP := func() {
		ip := ipPart()
		if ip == "" {
			setErrHint("")
			setContainerError(false)
			return
		}
		if net.ParseIP(ip) == nil {
			setErrHint("Invalid IP address")
			setContainerError(true)
			return
		}
		setErrHint("")
		setContainerError(false)
	}

	helpClass := "text-xs text-ink-4 mt-1.5"
	if helpText == "" {
		helpClass = "hidden"
	}

	ipAttrs := Attr{
		"id":          fieldID + "-ip",
		"class":       "flex-1 min-w-0 px-3.5 py-2.5 bg-transparent text-ink-1 text-sm placeholder-ink-4 focus:outline-none font-mono rounded-md",
		"type":        "text",
		"placeholder": placeholderIP,
	}
	if initIP != "" {
		ipAttrs["value"] = initIP
	}

	// Build right-side content: static /32 label or prefix dropdown + tooltip
	var rightSide []loom.Node
	if hostOnly {
		rightSide = []loom.Node{
			Span(Apply(Attr{"class": "text-ink-4 text-sm font-mono select-none shrink-0 pr-3"}), Text("/32")),
		}
	} else {
		selectChildren := []loom.Node{
			Apply(Attr{
				"id":    fieldID + "-pfx",
				"class": "bg-transparent text-ink-1 text-sm font-mono py-2.5 pr-2 pl-1 focus:outline-none cursor-pointer appearance-none",
			}),
			Apply(On{"change": func(evt *Event) {
				val := evt.Target().Get("value").String()
				setPrefixPart(val)
				syncValue()
				clearError()
			}}),
		}
		for _, opt := range cidrPrefixOptions {
			attrs := Attr{"value": opt.Value}
			if opt.Value == initPrefix {
				attrs["selected"] = "selected"
			}
			selectChildren = append(selectChildren, Elem("option", Apply(attrs), Text(opt.Label)))
		}
		rightSide = []loom.Node{
			Span(Apply(Attr{"class": "text-ink-4 text-sm select-none shrink-0 px-0.5"}), Text("/")),
			Elem("select", selectChildren...),
			Bind(func() loom.Node {
				pfx := prefixPart()
				hint := prefixHostHint(pfx)
				return Tooltip(
					Span(Apply(Attr{"class": "text-ink-4 hover:text-ink-2 transition-colors pr-2.5 pl-1"}), Icon("info", 13)),
					[]string{hint},
				)
			}),
		}
	}

	containerChildren := []loom.Node{
		Apply(Attr{
			"id":    fieldID,
			"class": "flex items-center bg-surface-0 border border-line-1 rounded-md focus-within:border-wg-600/40 focus-within:ring-1 focus-within:ring-wg-600/15 transition-colors",
		}),
		Input(
			Apply(ipAttrs),
			Apply(On{"input": func(evt *EventInput) {
				setIPPart(evt.InputValue())
				syncValue()
				clearError()
			}}),
			Apply(On{"blur": func() { validateIP() }}),
		),
	}
	containerChildren = append(containerChildren, rightSide...)

	children := []loom.Node{
		Apply(Attr{"class": "mb-4 overflow-visible"}),
		Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text(label)),
		Div(containerChildren...),
	}

	// Subnet summary (only for non-host-only mode)
	if !hostOnly {
		children = append(children, Bind(func() loom.Node {
			s := summaryText()
			cls := "text-[11px] text-ink-4 mt-1.5 font-mono opacity-0 h-0"
			txt := "\u00a0"
			if s != "" {
				cls = "text-[11px] text-ink-4 mt-1.5 font-mono"
				txt = s
			}
			return P(Apply(Attr{"class": cls}), Text(txt))
		}))
	}

	// Inline error hint
	children = append(children, Bind(func() loom.Node {
		hint := errHint()
		cls := "text-xs text-red-400 mt-1 opacity-0 h-0"
		txt := "\u00a0"
		if hint != "" {
			cls = "text-xs text-red-400 mt-1"
			txt = hint
		}
		return P(Apply(Attr{"class": cls}), Text(txt))
	}))

	children = append(children, P(Apply(Attr{"class": helpClass}), Text(helpText)))

	return Div(children...)
}

// TypeBadge renders a badge for the interface type.
func TypeBadge(ifaceType string) loom.Node {
	if ifaceType == "site-to-site" {
		return Badge("Site-to-Site", "amber")
	}
	return Badge("Client Access", "teal")
}

func PeerTypeBadge(peerType string) loom.Node {
	if peerType == "site" {
		return Badge("Site", "amber")
	}
	return Badge("Client", "teal")
}

// typeSelectorCard renders a clickable card for interface type selection.
func typeSelectorCard(iconName, title, description string, selected bool, onClick func()) loom.Node {
	cls := "flex items-start gap-3 p-4 rounded-lg border-2 cursor-pointer transition-all "
	if selected {
		cls += "border-wg-500/50 bg-wg-600/10"
	} else {
		cls += "border-line-1 hover:border-line-2 bg-surface-0"
	}
	iconCls := "mt-0.5 "
	if selected {
		iconCls += "text-wg-400"
	} else {
		iconCls += "text-ink-4"
	}
	titleCls := "text-sm font-semibold "
	if selected {
		titleCls += "text-wg-400"
	} else {
		titleCls += "text-ink-2"
	}
	return Div(
		Apply(Attr{"class": cls}),
		Apply(On{"click": func() { onClick() }}),
		Span(Apply(Attr{"class": iconCls}), Icon(iconName, 20)),
		Div(
			Span(Apply(Attr{"class": "block " + titleCls}), Text(title)),
			Span(Apply(Attr{"class": "block text-xs text-ink-4 mt-0.5"}), Text(description)),
		),
	)
}

// FormatBytes formats bytes to human-readable.
func formatListenAddr(endpoint string, port int) string {
	if endpoint != "" {
		return fmt.Sprintf("%s:%d", endpoint, port)
	}
	return fmt.Sprintf(":%d", port)
}

func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// FormatTimestamp converts an ISO/RFC3339 timestamp to US 12-hour format.
func FormatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return ts
		}
	}
	return t.Local().Format("Jan 2, 2006 3:04:05 PM")
}

// FormatDuration formats an RFC3339 timestamp as a human-readable duration from now.
func FormatDuration(since string) string {
	t, err := time.Parse(time.RFC3339, since)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return ""
		}
	}
	d := time.Now().Sub(t)
	if d < 0 {
		d = 0
	}
	return FormatSeconds(int64(d.Seconds()))
}

// parseUnixSince parses an RFC3339 string and returns the Unix timestamp as a string.
func parseUnixSince(since string) string {
	t, err := time.Parse(time.RFC3339, since)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return ""
		}
	}
	return strconv.FormatInt(t.Unix(), 10)
}

// UptimeSpan renders a span with data-since attribute for the JS-driven uptime timer.
func UptimeSpan(connectedSince, class string) loom.Node {
	unix := parseUnixSince(connectedSince)
	if unix == "" {
		// Identical DOM structure with hidden class to satisfy Loom Bind requirements
		return Span(
			Apply(Attr{"class": "hidden"}),
			Text(""),
		)
	}
	return Span(
		Apply(Attr{"class": class, "data-since": unix}),
		Text(FormatDuration(connectedSince)), // initial value, JS takes over
	)
}

// initUptimeTimers starts a global JS interval that updates all [data-since] elements every second.
func initUptimeTimers() {
	js.Global().Call("eval", `
(function() {
	function fmtUptime(s) {
		if (s < 0) s = 0;
		var d = Math.floor(s / 86400);
		var h = Math.floor((s % 86400) / 3600);
		var m = Math.floor((s % 3600) / 60);
		var sec = s % 60;
		if (d > 0) return h > 0 ? d + 'd ' + h + 'h' : d + 'd';
		if (h > 0) return m > 0 ? h + 'h ' + m + 'm' : h + 'h';
		if (m > 0) return sec > 0 ? m + 'm ' + sec + 's' : m + 'm';
		return sec + 's';
	}
	if (window._uptimeInterval) clearInterval(window._uptimeInterval);
	window._uptimeInterval = setInterval(function() {
		var now = Math.floor(Date.now() / 1000);
		var els = document.querySelectorAll('[data-since]');
		for (var i = 0; i < els.length; i++) {
			var since = parseInt(els[i].getAttribute('data-since'), 10);
			if (!isNaN(since)) {
				els[i].textContent = fmtUptime(now - since);
			}
		}
	}, 1000);
})();
`)
}

// FormatSeconds formats a duration in seconds as a human-readable string.
func FormatSeconds(totalSecs int64) string {
	if totalSecs <= 0 {
		return "0s"
	}

	days := totalSecs / 86400
	hours := (totalSecs % 86400) / 3600
	mins := (totalSecs % 3600) / 60
	secs := totalSecs % 60

	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	case mins > 0 && secs > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	case mins > 0:
		return fmt.Sprintf("%dm", mins)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// Toast notification
var (
	toastMsg    func() string
	setToastMsg func(string)
)

// Celebration toast
var (
	celebrationPeer    func() string
	setCelebrationPeer func(string)
)

func initToast() {
	toastMsg, setToastMsg = Signal("")
	celebrationPeer, setCelebrationPeer = Signal("")
}

func showToast(msg string) {
	setToastMsg(msg)
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		setToastMsg("")
		return nil
	}), 2000)
}

// Toast renders a small notification that appears at the bottom center.
func Toast() loom.Node {
	return Bind(func() loom.Node {
		msg := toastMsg()
		cls := "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-toast"
		if msg == "" {
			cls = "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none opacity-0"
			msg = "\u00a0" // non-breaking space to preserve structure
		}
		return Div(
			Apply(Attr{"class": cls}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2.5 px-5 py-3 bg-surface-3 border border-line-3 text-ink-1 text-sm font-medium rounded-lg"}),
				Icon("check", 16),
				Span(Text(msg)),
			),
		)
	})
}

// showCelebration displays the celebration toast and triggers confetti.
func showCelebration(name string) {
	setCelebrationPeer(name)
	triggerConfetti()
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		setCelebrationPeer("")
		return nil
	}), 5000)
}

// triggerConfetti injects CSS confetti pieces into the DOM.
func triggerConfetti() {
	doc := js.Global().Get("document")
	container := doc.Call("createElement", "div")
	container.Set("id", "confetti-container")
	doc.Get("body").Call("appendChild", container)

	colors := []string{"#f87171", "#34d399", "#60a5fa", "#fbbf24", "#a78bfa", "#f472b6", "#2dd4bf"}
	for i := 0; i < 40; i++ {
		piece := doc.Call("createElement", "div")
		piece.Get("classList").Call("add", "confetti-piece")
		piece.Get("style").Set("left", fmt.Sprintf("%d%%", i*100/40+1))
		piece.Get("style").Set("backgroundColor", colors[i%len(colors)])
		dur := 1.5 + float64(i%5)*0.4
		piece.Get("style").Set("animationDuration", fmt.Sprintf("%.1fs", dur))
		piece.Get("style").Set("animationDelay", fmt.Sprintf("%.2fs", float64(i%8)*0.1))
		container.Call("appendChild", piece)
	}

	// Remove confetti container after animations complete
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		container.Call("remove")
		return nil
	}), 5000)
}

// CelebrationToast renders a celebration notification for first peer connection.
func CelebrationToast() loom.Node {
	return Bind(func() loom.Node {
		name := celebrationPeer()
		cls := "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-celebrate"
		msg := fmt.Sprintf("\U0001f389  %s is connected!", name)
		if name == "" {
			cls = "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none opacity-0"
			msg = "\u00a0"
		}
		return Div(
			Apply(Attr{"class": cls}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2.5 px-5 py-3 bg-green-600/70 backdrop-blur-md border border-green-500/50 text-green-200 text-sm font-medium rounded-lg shadow-lg"}),
				Span(Text(msg)),
			),
		)
	})
}

// confirmModalState holds the state for the custom confirmation modal.
type confirmModalState struct {
	Message   string
	OnConfirm func()
}

// Package-level signals for the confirm modal.
var (
	confirmState    func() *confirmModalState
	setConfirmState func(*confirmModalState)
)

func initConfirmModal() {
	confirmState, setConfirmState = Signal[*confirmModalState](nil)
}

// ConfirmAction shows a custom modal with the given message.
func ConfirmAction(message string, onConfirm func()) {
	setConfirmState(&confirmModalState{
		Message:   message,
		OnConfirm: onConfirm,
	})
}

func dismissConfirm() {
	setConfirmState(nil)
}

// ConfirmModal renders the confirmation modal overlay.
func ConfirmModal() loom.Node {
	return Bind(func() loom.Node {
		st := confirmState()
		visible := st != nil
		msg := ""
		if visible {
			msg = st.Message
		}

		backdropClass := "fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in"
		cardClass := "bg-surface-2 border border-line-1 rounded-lg max-w-sm w-full mx-4 overflow-hidden animate-scale-in"
		if !visible {
			backdropClass = "fixed inset-0 z-50 hidden"
			cardClass = "bg-surface-2 border border-line-1 rounded-lg max-w-sm w-full mx-4 overflow-hidden hidden"
		}

		return Div(
			Apply(Attr{"class": backdropClass}),
			Apply(On{"click": func() { dismissConfirm() }}),
			Div(
				Apply(Attr{"class": cardClass}),
				Apply(On{"click": func(evt *Event) { evt.StopPropagation() }}),
				// Status stripe — danger
				Div(Apply(Attr{"class": "h-[2px] bg-red-600"})),
				Div(
					Apply(Attr{"class": "px-6 pt-5 pb-0"}),
					Div(
						Apply(Attr{"class": "flex items-start gap-4"}),
						Div(
							Apply(Attr{"class": "flex-shrink-0 w-9 h-9 rounded-md bg-red-500/10 flex items-center justify-center"}),
							Span(Apply(Attr{"class": "text-red-400"}), Apply(innerHTML(icons["triangle-alert"](16)))),
						),
						Div(
							Apply(Attr{"class": "flex-1 min-w-0"}),
							P(Apply(Attr{"class": "text-sm font-semibold text-ink-1"}), Text("Confirm action")),
							P(Apply(Attr{"class": "mt-1.5 text-sm text-ink-3 leading-relaxed"}), Text(msg)),
						),
					),
				),
				Div(
					Apply(Attr{"class": "flex items-center justify-end gap-3 px-6 py-4 mt-2"}),
					Button(
						Apply(Attr{"class": "inline-flex items-center justify-center text-xs font-medium rounded-md transition-all duration-100 cursor-pointer px-4 py-2 text-ink-3 hover:bg-surface-3 hover:text-ink-1"}),
						Apply(On{"click": func() { dismissConfirm() }}),
						Text("Cancel"),
					),
					Button(
						Apply(Attr{"class": "inline-flex items-center justify-center text-xs font-semibold rounded-md transition-all duration-100 cursor-pointer px-4 py-2 bg-red-600 text-white hover:bg-red-700"}),
						Apply(On{"click": func() {
							st := confirmState()
							if st != nil && st.OnConfirm != nil {
								cb := st.OnConfirm
								dismissConfirm()
								cb()
							} else {
								dismissConfirm()
							}
						}}),
						Text("Confirm"),
					),
				),
			),
		)
	})
}

// Spinner renders a CSS-animated loading spinner.
func Spinner() loom.Node {
	return Div(
		Apply(Attr{"class": "flex items-center justify-center p-8"}),
		Div(Apply(Attr{"class": "spinner"})),
	)
}

// LoadingView renders a spinner that hides when loading becomes false.
func LoadingView(loading Accessor[bool]) loom.Node {
	return Bind(func() loom.Node {
		if loading() {
			return Div(
				Apply(Attr{"class": "flex items-center justify-center p-16"}),
				Div(Apply(Attr{"class": "spinner"})),
			)
		}
		return Div(
			Apply(Attr{"class": "hidden"}),
			Div(Apply(Attr{"class": "hidden"})),
		)
	})
}

// EmptyState renders a centered message for empty lists.
func EmptyState(msg string) loom.Node {
	lower := strings.ToLower(msg)
	iconName := "inbox"
	if strings.Contains(lower, "log") || strings.Contains(lower, "connection") {
		iconName = "scroll-text"
	} else if strings.Contains(lower, "interface") {
		iconName = "chevrons-left-right-ellipsis"
	} else if strings.Contains(lower, "peer") {
		iconName = "network"
	} else if strings.Contains(lower, "user") {
		iconName = "users"
	}
	return Div(
		Apply(Attr{"class": "flex flex-col items-center justify-center py-20 gap-4"}),
		Span(Apply(Attr{"class": "text-ink-4"}), Icon(iconName, 32)),
		P(Apply(Attr{"class": "text-ink-3 text-sm"}), Text(msg)),
	)
}

// RouteView conditionally renders a view.
func RouteView(when func() bool, fn func() loom.Node) loom.Node {
	var cached loom.Node
	return Bind(func() loom.Node {
		if !when() {
			cached = nil
			return Div(Apply(Attr{"class": "hidden"}))
		}
		if cached == nil {
			cached = fn()
		}
		return Div(cached)
	})
}

// ErrorAlert renders an error message with optional technical details disclosure.
func ErrorAlert(errInfo Accessor[ErrorInfo]) loom.Node {
	return Bind(func() loom.Node {
		info := errInfo()

		// Single return — compute all values up front, toggle via CSS classes
		wrapperCls := "hidden"
		iconSVG := ""
		msg := ""
		detailCls := "hidden"
		detailText := ""

		if info.Message != "" {
			wrapperCls = "mb-5 p-3.5 bg-red-500/10 border border-red-500/20 rounded-md text-red-400 text-sm"
			iconSVG = icons["triangle-alert"](15)
			msg = info.Message
			if info.Detail != "" {
				detailCls = "mt-2 pt-2 border-t border-red-500/20"
				detailText = info.Detail
			}
		}

		return Div(
			Apply(Attr{"class": wrapperCls}),
			Div(
				Apply(Attr{"class": "flex items-center gap-3"}),
				Span(Apply(Attr{"class": "shrink-0 text-red-400"}), Apply(innerHTML(iconSVG))),
				Span(Text(msg)),
			),
			Elem("details",
				Apply(Attr{"class": detailCls}),
				Elem("summary",
					Apply(Attr{"class": "cursor-pointer text-red-400/70 hover:text-red-400 select-none text-xs"}),
					Text("Technical details"),
				),
				Elem("pre",
					Apply(Attr{"class": "mt-1 text-[11px] font-mono text-red-400/60 whitespace-pre-wrap break-all"}),
					Text(detailText),
				),
			),
		)
	})
}
