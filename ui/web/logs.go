//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

var logsPollInterval js.Value

func LogsView(initialIfaceID string) loom.Node {
	logs, setLogs := Signal[[]connectionLogData](nil)
	loading, setLoading := Signal(true)
	ifaces, setIfaces := Signal[[]interfaceData](nil)
	activeIface, setActiveIface := Signal("")

	loadLogs := func(ifaceID string) {
		if ifaceID == "" {
			setLogs(nil)
			setLoading(false)
			return
		}
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/interfaces/"+ifaceID+"/logs?limit=100", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []connectionLogData
			json.Unmarshal(resp.Data, &list)
			setLogs(list)
			setLoading(false)
		}()
	}

	startPolling := func() {
		if !logsPollInterval.IsUndefined() && !logsPollInterval.IsNull() {
			js.Global().Call("clearInterval", logsPollInterval)
		}
		logsPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
			if id := activeIface(); id != "" {
				loadLogs(id)
			}
			return nil
		}), 5000)
	}

	// Load interfaces list
	Effect(func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/interfaces", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []interfaceData
			json.Unmarshal(resp.Data, &list)
			setIfaces(list)

			// Use URL param if provided, otherwise redirect to first interface
			selected := initialIfaceID
			if selected == "" && len(list) > 0 {
				selected = list[0].ID
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/logs/"+selected)
			}
			if selected != "" {
				setActiveIface(selected)
				loadLogs(selected)
				startPolling()
			} else {
				setLoading(false)
			}
		}()
	})

	// Interface selector dropdown
	ifaceSelector := Show(func() bool { return len(ifaces()) > 0 }, func() loom.Node {
		ifList := ifaces()
		opts := make([]loom.Node, 0, len(ifList))
		for _, iface := range ifList {
			attrs := Attr{"value": iface.ID}
			if iface.ID == initialIfaceID || (initialIfaceID == "" && len(opts) == 0) {
				attrs["selected"] = "selected"
			}
			opts = append(opts, Elem("option", Apply(attrs), Text(iface.ID)))
		}
		return Elem("select",
			Apply(Attr{"class": "px-4 py-2 bg-surface-1 border border-line-2 rounded-lg text-sm text-ink-1 focus:outline-none focus:border-wg-600/50 focus:ring-1 focus:ring-wg-600/20 transition-colors"}),
			Apply(On{"change": func(evt *EventInput) {
				val := evt.InputValue()
				setActiveIface(val)
				setLoading(true)
				loadLogs(val)
				startPolling()
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/logs/"+val)
			}}),
			Fragment(opts...),
		)
	})

	return Div(
		PageHeader("Connection Logs", "Real-time connection activity", ifaceSelector),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			return Bind(func() loom.Node {
				list := logs()
				if len(list) == 0 {
					return EmptyState("No connection logs yet")
				}

				cards := make([]loom.Node, 0, len(list))
				for _, log := range list {
					log := log
					eventColor := ""
					if log.Event == "connected" {
						eventColor = "emerald"
					}
					peerLabel := log.PeerID[:8] + "..."
					if log.PeerName != "" {
						peerLabel = log.PeerName
					}

					cards = append(cards, Div(
						Apply(Attr{"class": "bg-surface-1 rounded-lg px-5 py-4"}),
						// Row: badge + peer info + stats
						Div(
							Apply(Attr{"class": "flex items-start gap-3"}),
							// Badge
							Div(Apply(Attr{"class": "flex-shrink-0 mt-0.5"}), Badge(log.Event, eventColor)),
							// Peer info — grows to fill
							Div(
								Apply(Attr{"class": "flex-1 min-w-0"}),
								Div(
									Apply(Attr{"class": "flex items-center gap-2"}),
									Span(Apply(Attr{"class": "text-sm text-ink-1 font-medium"}), Text(peerLabel)),
									func() loom.Node {
										if log.Endpoint != "" {
											return Span(Apply(Attr{"class": "font-mono text-[11px] text-ink-4"}), Text(log.Endpoint))
										}
										return Span()
									}(),
								),
								Div(Apply(Attr{"class": "text-[11px] text-ink-4 mt-0.5"}), Text(FormatTimestamp(log.RecordedAt))),
								// Mobile transfer stats
								Div(
									Apply(Attr{"class": "sm:hidden font-mono text-xs text-ink-3 mt-1.5 flex items-center gap-3"}),
									Span(Text(fmt.Sprintf("↓%s", FormatBytes(log.TransferRx)))),
									Span(Text(fmt.Sprintf("↑%s", FormatBytes(log.TransferTx)))),
								),
							),
							// Desktop transfer stats
							Div(
								Apply(Attr{"class": "hidden sm:flex font-mono text-xs text-ink-3 items-center gap-3 flex-shrink-0"}),
								Span(Text(fmt.Sprintf("↓%s", FormatBytes(log.TransferRx)))),
								Span(Text(fmt.Sprintf("↑%s", FormatBytes(log.TransferTx)))),
							),
						),
					))
				}

				return Div(
					Apply(Attr{"class": "space-y-2"}),
					Fragment(cards...),
				)
			})
		}),
	)
}
