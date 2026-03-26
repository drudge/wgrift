//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Card wraps content in a white bordered panel.
func Card(children ...loom.Node) loom.Node {
	return Div(
		append([]loom.Node{
			Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-5"}),
		}, children...)...,
	)
}

// CardHeader is a card title row.
func CardHeader(title string, extra ...loom.Node) loom.Node {
	children := []loom.Node{
		Apply(Attr{"class": "flex items-center justify-between mb-4"}),
		H3(Apply(Attr{"class": "text-xs font-medium text-gray-400 uppercase tracking-widest"}), Text(title)),
	}
	children = append(children, extra...)
	return Div(children...)
}

// Btn renders a button with optional variant.
func Btn(label string, variant string, handler func()) loom.Node {
	class := "inline-flex items-center justify-center text-sm font-medium rounded-md border transition-colors cursor-pointer "
	switch variant {
	case "primary":
		class += "px-4 py-2 bg-teal-600 border-teal-600 text-white hover:bg-teal-700 active:bg-teal-800"
	case "danger":
		class += "px-3 py-1.5 text-red-600 border-red-200 hover:bg-red-50 hover:border-red-300"
	case "ghost":
		class += "px-3 py-1.5 border-gray-300 text-gray-600 hover:bg-gray-50 hover:text-gray-900"
	default:
		class += "px-3 py-1.5 border-gray-300 text-gray-600 hover:bg-gray-50"
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
			"class": "w-8 h-8 rounded-md flex items-center justify-center text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors",
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
			"class": "w-8 h-8 rounded-md flex items-center justify-center text-gray-400 hover:text-red-600 hover:bg-red-50 transition-colors",
			"title": title,
		}),
		Apply(On{"click": func() { handler() }}),
		Icon(iconName, 15),
	)
}

// StatusDot renders a colored status indicator.
// Three states: disabled (red ring with line), disconnected (gray), connected (green pulse).
func StatusDot(enabled, connected bool) loom.Node {
	if !enabled {
		// Disabled: red outline ring with a diagonal line through it
		return Span(Apply(Attr{"class": "inline-flex items-center justify-center w-3 h-3 rounded-full border-[1.5px] border-red-400 bg-white"}),
			Span(Apply(Attr{"class": "block w-[7px] h-[1.5px] bg-red-400 rounded-full -rotate-45"})),
		)
	}
	if connected {
		return Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 status-pulse"}))
	}
	return Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-gray-300"}))
}

// Badge renders a small label.
func Badge(label string, color string) loom.Node {
	class := "inline-block px-2 py-0.5 text-xs font-medium rounded-md "
	switch color {
	case "teal":
		class += "bg-teal-50 text-teal-700 border border-teal-200"
	case "amber":
		class += "bg-amber-50 text-amber-700 border border-amber-200"
	case "red":
		class += "bg-red-50 text-red-700 border border-red-200"
	case "emerald":
		class += "bg-emerald-50 text-emerald-700 border border-emerald-200"
	default:
		class += "bg-gray-100 text-gray-600 border border-gray-200"
	}
	return Span(Apply(Attr{"class": class}), Text(label))
}

// MonoText renders text in monospace.
func MonoText(content string) loom.Node {
	return Span(Apply(Attr{"class": "font-mono text-sm"}), Text(content))
}

// FormField renders a label + input.
// The value accessor is read once for the initial value (not reactive, to avoid cursor reset).
func FormField(label, inputType, placeholder string, value Accessor[string], onInput func(string)) loom.Node {
	initVal := value()
	attrs := Attr{
		"class":       "w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-gray-900 text-sm placeholder-gray-400 focus:outline-none focus:border-teal-500 focus:ring-1 focus:ring-teal-500/20",
		"type":        inputType,
		"placeholder": placeholder,
	}
	if initVal != "" {
		attrs["value"] = initVal
	}
	return Div(
		Apply(Attr{"class": "mb-4"}),
		Elem("label", Apply(Attr{"class": "block text-sm text-gray-600 mb-1.5"}), Text(label)),
		Input(
			Apply(attrs),
			Apply(On{"input": func(evt *EventInput) {
				onInput(evt.InputValue())
			}}),
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
		// Try with nanoseconds
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
			Apply(Attr{"class": "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-fade-in"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2 px-4 py-2.5 bg-gray-900 text-white text-sm font-medium rounded-lg shadow-lg"}),
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

// initConfirmModal sets up the confirm modal signals. Call from main before Loom mounts.
func initConfirmModal() {
	confirmState, setConfirmState = Signal[*confirmModalState](nil)
}

// ConfirmAction shows a custom modal with the given message.
// If the user clicks "Confirm", onConfirm is called.
func ConfirmAction(message string, onConfirm func()) {
	setConfirmState(&confirmModalState{
		Message:   message,
		OnConfirm: onConfirm,
	})
}

// dismissConfirm closes the modal without running the callback.
func dismissConfirm() {
	setConfirmState(nil)
}

// ConfirmModal renders the confirmation modal overlay.
// Place this once in the layout so it's always available.
// Uses Bind with identical DOM structure in both branches (hidden vs visible)
// to avoid Loom's Show cleanup bug.
func ConfirmModal() loom.Node {
	return Bind(func() loom.Node {
		st := confirmState()
		visible := st != nil
		msg := ""
		if visible {
			msg = st.Message
		}

		// Always render the same structure; toggle visibility via classes
		backdropClass := "fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px] animate-fade-in"
		cardClass := "bg-white rounded-xl shadow-2xl shadow-black/10 max-w-sm w-full mx-4 overflow-hidden animate-scale-in"
		if !visible {
			backdropClass = "fixed inset-0 z-50 hidden"
			cardClass = "bg-white rounded-xl shadow-2xl max-w-sm w-full mx-4 overflow-hidden hidden"
		}

		return Div(
			// Backdrop
			Apply(Attr{"class": backdropClass}),
			Apply(On{"click": func() { dismissConfirm() }}),
			// Modal card
			Div(
				Apply(Attr{"class": cardClass}),
				Apply(On{"click": func(evt *Event) { evt.StopPropagation() }}),
				// Header with icon + title
				Div(
					Apply(Attr{"class": "px-5 pt-5 pb-0"}),
					Div(
						Apply(Attr{"class": "flex items-start gap-3"}),
						// Icon
						Div(
							Apply(Attr{"class": "flex-shrink-0 w-9 h-9 rounded-lg bg-red-50 border border-red-100 flex items-center justify-center"}),
							Span(Apply(Attr{"class": "text-red-500"}), Apply(innerHTML(icons["triangle-alert"](18)))),
						),
						// Title + message
						Div(
							Apply(Attr{"class": "flex-1 min-w-0"}),
							P(Apply(Attr{"class": "text-sm font-semibold text-gray-900"}), Text("Are you sure?")),
							P(Apply(Attr{"class": "mt-1 text-sm text-gray-500 leading-relaxed"}), Text(msg)),
						),
					),
				),
				// Buttons
				Div(
					Apply(Attr{"class": "flex items-center justify-end gap-2 px-5 py-4 mt-3 bg-gray-50 border-t border-gray-100"}),
					Button(
						Apply(Attr{"class": "inline-flex items-center justify-center text-xs font-medium rounded-md border transition-colors cursor-pointer px-3 py-1.5 bg-white border-gray-200 text-gray-600 hover:bg-gray-50 hover:text-gray-900 shadow-sm"}),
						Apply(On{"click": func() { dismissConfirm() }}),
						Text("Cancel"),
					),
					Button(
						Apply(Attr{"class": "inline-flex items-center justify-center text-xs font-medium rounded-md border transition-colors cursor-pointer px-3 py-1.5 bg-red-600 border-red-600 text-white hover:bg-red-700 active:bg-red-800 shadow-sm"}),
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
				Apply(Attr{"class": "flex items-center justify-center p-8"}),
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
	return Div(
		Apply(Attr{"class": "flex flex-col items-center justify-center py-12 gap-3 text-gray-300"}),
		Icon("network", 32),
		P(Apply(Attr{"class": "text-gray-400 text-sm"}), Text(msg)),
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
			Apply(Attr{"class": "mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm flex items-center gap-2"}),
			Span(Apply(Attr{"class": "shrink-0 text-red-500"}), Apply(innerHTML(icons["triangle-alert"](16)))),
			Span(Text(msg)),
		)
	})
}
