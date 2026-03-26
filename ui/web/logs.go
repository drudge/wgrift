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

	return Div(
		Div(
			Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-8"}),
			H2(Apply(Attr{"class": "text-xl font-semibold text-gray-900"}), Text("Connection Logs")),

			// Interface selector
			Show(func() bool { return len(ifaces()) > 0 }, func() loom.Node {
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
					Apply(Attr{"class": "px-3 py-1.5 bg-white border border-gray-300 rounded-md text-sm text-gray-700"}),
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
			}),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			return Bind(func() loom.Node {
				list := logs()
				if len(list) == 0 {
					return EmptyState("No connection logs yet")
				}

				cards := make([]loom.Node, 0, len(list))
				rows := make([]loom.Node, 0, len(list))
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

					// Mobile card
					cards = append(cards, Div(
						Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg px-4 py-3"}),
						Div(
							Apply(Attr{"class": "flex items-center justify-between mb-1.5"}),
							Div(
								Apply(Attr{"class": "flex items-center gap-2"}),
								Badge(log.Event, eventColor),
								Span(Apply(Attr{"class": "text-sm text-gray-700"}), Text(peerLabel)),
							),
							Span(Apply(Attr{"class": "font-mono text-[11px] text-gray-400"}), Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(log.TransferRx), FormatBytes(log.TransferTx)))),
						),
						Div(Apply(Attr{"class": "text-[11px] text-gray-400"}), Text(FormatTimestamp(log.RecordedAt))),
					))

					// Desktop row
					rows = append(rows, Elem("tr",
						Apply(Attr{"class": "border-b border-gray-100"}),
						Elem("td", Apply(Attr{"class": "px-4 py-2 text-xs text-gray-400"}), Text(FormatTimestamp(log.RecordedAt))),
						Elem("td", Apply(Attr{"class": "px-4 py-2"}), Badge(log.Event, eventColor)),
						Elem("td", Apply(Attr{"class": "px-4 py-2"}),
							Div(
								Span(Apply(Attr{"class": "text-sm text-gray-700"}), Text(peerLabel)),
								Span(Apply(Attr{"class": "ml-2 font-mono text-xs text-gray-400"}), Text(log.PeerID[:8])),
							),
						),
						Elem("td", Apply(Attr{"class": "px-4 py-2 font-mono text-xs text-gray-400"}), Text(FormatBytes(log.TransferRx))),
						Elem("td", Apply(Attr{"class": "px-4 py-2 font-mono text-xs text-gray-400"}), Text(FormatBytes(log.TransferTx))),
					))
				}

				return Div(
					// Mobile cards
					Div(
						Apply(Attr{"class": "md:hidden space-y-2"}),
						Fragment(cards...),
					),
					// Desktop table
					Div(
						Apply(Attr{"class": "hidden md:block bg-white border border-gray-200 rounded-lg overflow-hidden"}),
						Elem("table",
							Apply(Attr{"class": "w-full text-sm"}),
							Elem("thead",
								Elem("tr",
									Apply(Attr{"class": "border-b border-gray-200 text-left text-xs uppercase tracking-wider text-gray-400"}),
									Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Time")),
									Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Event")),
									Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Peer")),
									Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("RX")),
									Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("TX")),
								),
							),
							Elem("tbody", rows...),
						),
					),
				)
			})
		}),
	)
}
