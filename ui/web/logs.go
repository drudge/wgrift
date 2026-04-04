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

var (
	logsPollInterval js.Value
	cachedLogs       []connectionLogData
	cachedLogsIfaces []interfaceData
	cachedLogsIface  string // active interface ID
)

func LogsView(initialIfaceID string) loom.Node {
	// Set up polling — each tick fetches fresh data then does a clean remount
	// via refreshRoute, which avoids Bind's DOM reconciliation issues with
	// variable-length lists (log entries changing between polls).
	if !logsPollInterval.IsUndefined() && !logsPollInterval.IsNull() {
		js.Global().Call("clearInterval", logsPollInterval)
	}
	logsPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
		if cachedLogsIface != "" {
			go func() {
				var resp apiResponse
				if err := apiFetch("GET", "/api/v1/interfaces/"+cachedLogsIface+"/logs?limit=100", nil, &resp); err != nil {
					return
				}
				var list []connectionLogData
				if err := json.Unmarshal(resp.Data, &list); err == nil {
					cachedLogs = list
					refreshRoute()
				}
			}()
		}
		return nil
	}), 5000)

	// First load — no cached interfaces yet, fetch everything
	if cachedLogsIfaces == nil {
		loading, setLoading := Signal(true)
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/interfaces", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []interfaceData
			json.Unmarshal(resp.Data, &list)
			cachedLogsIfaces = list

			// Use URL param if provided, otherwise use first interface
			selected := initialIfaceID
			if selected == "" && len(list) > 0 {
				selected = list[0].ID
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/logs/"+selected)
			}

			if selected != "" {
				cachedLogsIface = selected
				var logsResp apiResponse
				if err := apiFetch("GET", "/api/v1/interfaces/"+selected+"/logs?limit=100", nil, &logsResp); err != nil {
					setLoading(false)
					return
				}
				var logs []connectionLogData
				json.Unmarshal(logsResp.Data, &logs)
				cachedLogs = logs
			}
			refreshRoute()
		}()
		return Div(LoadingView(loading))
	}

	// Use initialIfaceID to handle route changes (e.g. switching interface via dropdown)
	if initialIfaceID != "" && initialIfaceID != cachedLogsIface {
		cachedLogsIface = initialIfaceID
		cachedLogs = nil
		loading, setLoading := Signal(true)
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/interfaces/"+initialIfaceID+"/logs?limit=100", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []connectionLogData
			json.Unmarshal(resp.Data, &list)
			cachedLogs = list
			refreshRoute()
		}()
		return Div(LoadingView(loading))
	}

	// Interface selector dropdown
	var ifaceSelector loom.Node
	if len(cachedLogsIfaces) > 0 {
		opts := make([]loom.Node, 0, len(cachedLogsIfaces))
		for _, iface := range cachedLogsIfaces {
			attrs := Attr{"value": iface.ID}
			if iface.ID == cachedLogsIface {
				attrs["selected"] = "selected"
			}
			opts = append(opts, Elem("option", Apply(attrs), Text(iface.ID)))
		}
		ifaceSelector = Elem("select",
			Apply(Attr{"class": "px-4 py-2 bg-surface-1 border border-line-2 rounded-lg text-sm text-ink-1 focus:outline-none focus:border-wg-600/50 focus:ring-1 focus:ring-wg-600/20 transition-colors"}),
			Apply(On{"change": func(evt *EventInput) {
				val := evt.InputValue()
				cachedLogsIface = val
				cachedLogs = nil
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/logs/"+val)
				refreshRoute()
			}}),
			Fragment(opts...),
		)
	} else {
		ifaceSelector = Span()
	}

	// Build log cards from cached data
	var content loom.Node
	if len(cachedLogs) == 0 {
		content = EmptyState("No connection logs yet")
	} else {
		cards := make([]loom.Node, 0, len(cachedLogs))
		for _, log := range cachedLogs {
			log := log
			eventColor := ""
			if log.Event == "connected" {
				eventColor = "emerald"
			}
			peerLabel := log.PeerID[:8] + "..."
			if log.PeerName != "" {
				peerLabel = log.PeerName
			}

			var endpointNode loom.Node
			if log.Endpoint != "" {
				endpointNode = Span(Apply(Attr{"class": "font-mono text-[11px] text-ink-4"}), Text(log.Endpoint))
			} else {
				endpointNode = Span()
			}

			var durationNode loom.Node
			if log.Duration > 0 {
				durationNode = Span(
					Apply(Attr{"class": "text-ink-3"}),
					Text(fmt.Sprintf("(%s)", FormatSeconds(log.Duration))),
				)
			} else {
				durationNode = Span()
			}

			cards = append(cards, Div(
				Apply(Attr{"class": "bg-surface-1 rounded-lg px-5 py-4"}),
				Div(
					Apply(Attr{"class": "flex items-start gap-3"}),
					Div(Apply(Attr{"class": "flex-shrink-0 mt-0.5"}), Badge(log.Event, eventColor)),
					Div(
						Apply(Attr{"class": "flex-1 min-w-0"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2"}),
							Span(Apply(Attr{"class": "text-sm text-ink-1 font-medium"}), Text(peerLabel)),
							endpointNode,
						),
						Div(
							Apply(Attr{"class": "flex items-center gap-2 text-[11px] text-ink-4 mt-0.5"}),
							Span(Text(FormatTimestamp(log.RecordedAt))),
							durationNode,
						),
						Div(
							Apply(Attr{"class": "sm:hidden font-mono text-xs text-ink-3 mt-1.5 flex items-center gap-3"}),
							Span(Text(fmt.Sprintf("↓%s", FormatBytes(log.TransferRx)))),
							Span(Text(fmt.Sprintf("↑%s", FormatBytes(log.TransferTx)))),
						),
					),
					Div(
						Apply(Attr{"class": "hidden sm:flex font-mono text-xs text-ink-3 items-center gap-3 flex-shrink-0"}),
						Span(Text(fmt.Sprintf("↓%s", FormatBytes(log.TransferRx)))),
						Span(Text(fmt.Sprintf("↑%s", FormatBytes(log.TransferTx)))),
					),
				),
			))
		}
		content = Div(
			Apply(Attr{"class": "space-y-2"}),
			Fragment(cards...),
		)
	}

	return Div(
		Div(
			Apply(Attr{"class": "flex items-end justify-between gap-4 mb-8"}),
			Div(
				H2(Apply(Attr{"class": "text-2xl font-bold text-ink-1 tracking-tight"}), Text("Connection Logs")),
				P(Apply(Attr{"class": "text-sm text-ink-3 mt-1 hidden sm:block"}), Text("Real-time connection activity")),
			),
			ifaceSelector,
		),
		content,
	)
}
