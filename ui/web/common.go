//go:build js && wasm

package main

import (
	"fmt"
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

// Toast notification
var (
	toastMsg    func() string
	setToastMsg func(string)
)

func initToast() {
	toastMsg, setToastMsg = Signal("")
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
		if msg == "" {
			return Div(Apply(Attr{"class": "hidden"}))
		}
		return Div(
			Apply(Attr{"class": "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-toast"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2.5 px-5 py-3 bg-surface-3 border border-line-3 text-ink-1 text-sm font-medium rounded-lg"}),
				Icon("check", 16),
				Text(msg),
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

// ErrorAlert renders an error message.
func ErrorAlert(errMsg Accessor[string]) loom.Node {
	return Bind(func() loom.Node {
		msg := errMsg()
		if msg == "" {
			return Div(
				Apply(Attr{"class": "hidden"}),
				Span(Apply(Attr{"class": "shrink-0"}), Apply(innerHTML(""))),
				Span(Text("")),
			)
		}
		return Div(
			Apply(Attr{"class": "mb-5 p-3.5 bg-red-500/10 border border-red-500/20 rounded-md text-red-400 text-sm flex items-center gap-3"}),
			Span(Apply(Attr{"class": "shrink-0 text-red-400"}), Apply(innerHTML(icons["triangle-alert"](15)))),
			Span(Text(msg)),
		)
	})
}
