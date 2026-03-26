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

var dashboardPollInterval js.Value

func DashboardView() loom.Node {
	data, setData := Signal[*dashboardData](nil)
	loading, setLoading := Signal(true)

	loadData := func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/dashboard", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var d dashboardData
			if err := json.Unmarshal(resp.Data, &d); err == nil {
				setData(&d)
			}
			setLoading(false)
		}()
	}

	Effect(func() {
		loadData()
		if !dashboardPollInterval.IsUndefined() && !dashboardPollInterval.IsNull() {
			js.Global().Call("clearInterval", dashboardPollInterval)
		}
		dashboardPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
			loadData()
			return nil
		}), 5000)
	})

	return Div(
		Div(
			Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-8"}),
			H2(Apply(Attr{"class": "text-xl font-semibold text-gray-900"}), Text("Status")),
			Btn("Refresh", "ghost", func() { refreshRoute() }),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			return Bind(func() loom.Node {
				d := data()
				if d == nil {
					return EmptyState("Unable to load status data")
				}

				runningCount := 0
				for _, iface := range d.Interfaces {
					if iface.Running {
						runningCount++
					}
				}

				ifaceNodes := make([]loom.Node, 0)
				if len(d.Interfaces) > 0 {
					ifaceNodes = interfaceCards(d.Interfaces)
				}

				return Div(
					// Summary stats
					Div(
						Apply(Attr{"class": "grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8"}),
						statCard("Interfaces", fmt.Sprintf("%d / %d running", runningCount, len(d.Interfaces))),
						statCard("Total Peers", fmt.Sprintf("%d", d.TotalPeers)),
						statCard("Active", fmt.Sprintf("%d", d.ActivePeers)),
						statCard("Transfer", fmt.Sprintf("%s / %s", FormatBytes(d.TotalRx), FormatBytes(d.TotalTx))),
					),

					// Interface cards
					Div(
						Div(
							Apply(Attr{"class": "flex items-center justify-between mb-4"}),
							H3(Apply(Attr{"class": "text-xs font-medium text-gray-400 uppercase tracking-widest"}), Text("Interfaces")),
							Btn("New Interface", "ghost", func() { navigate("/interfaces?action=create") }),
						),
						func() loom.Node {
							if len(ifaceNodes) == 0 {
								return EmptyState("No interfaces configured")
							}
							return Div(
								Apply(Attr{"class": "space-y-3"}),
								Fragment(ifaceNodes...),
							)
						}(),
					),
				)
			})
		}),
	)
}

func statCard(label, value string) loom.Node {
	return Div(
		Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-4"}),
		Div(Apply(Attr{"class": "text-[11px] text-gray-400 uppercase tracking-widest mb-1.5"}), Text(label)),
		Div(Apply(Attr{"class": "text-lg font-semibold text-gray-900 font-mono"}), Text(value)),
	)
}

func interfaceCards(ifaces []interfaceSummaryData) []loom.Node {
	nodes := make([]loom.Node, 0, len(ifaces))
	for _, iface := range ifaces {
		iface := iface
		nodes = append(nodes, interfaceCard(iface))
	}
	return nodes
}

func interfaceCard(iface interfaceSummaryData) loom.Node {
	statusColor := "bg-gray-300"
	statusText := "Stopped"
	statusClass := "text-gray-400"
	if iface.Running {
		statusColor = "bg-emerald-500 status-pulse"
		statusText = "Running"
		statusClass = "text-emerald-600"
	} else if iface.Enabled {
		statusColor = "bg-amber-400"
		statusText = "Enabled"
		statusClass = "text-amber-600"
	}

	return Div(
		Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-5"}),

		// Header: name + status
		Div(
			Apply(Attr{"class": "flex items-center justify-between mb-3"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-3"}),
				Span(Apply(Attr{"class": fmt.Sprintf("inline-block w-2.5 h-2.5 rounded-full %s", statusColor)})),
				Span(Apply(Attr{"class": "font-mono text-base font-semibold text-gray-900"}), Text(iface.ID)),
			),
			Span(Apply(Attr{"class": fmt.Sprintf("text-xs font-medium %s", statusClass)}), Text(statusText)),
		),

		// Details line
		Div(
			Apply(Attr{"class": "text-sm text-gray-400 mb-4 font-mono"}),
			Text(fmt.Sprintf("%s · port %d", iface.Address, iface.ListenPort)),
		),

		// Stats + Actions
		Div(
			Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t border-gray-100"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-4 text-sm"}),
				Span(
					Apply(Attr{"class": "text-gray-500"}),
					Span(Apply(Attr{"class": "text-emerald-600 font-mono font-medium"}), Text(fmt.Sprintf("%d", iface.ConnectedPeers))),
					Text(fmt.Sprintf(" / %d peers", iface.TotalPeers)),
				),
				func() loom.Node {
					if iface.TotalRx > 0 || iface.TotalTx > 0 {
						return Span(
							Apply(Attr{"class": "text-gray-400 text-xs font-mono"}),
							Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(iface.TotalRx), FormatBytes(iface.TotalTx))),
						)
					}
					return Span()
				}(),
			),
			Div(
				Apply(Attr{"class": "flex flex-wrap items-center gap-2"}),
				func() loom.Node {
					if iface.Running {
						return Fragment(
							Btn("Restart", "ghost", func() {
								go func() {
									apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/restart", iface.ID), nil, nil)
									refreshRoute()
								}()
							}),
							Btn("Stop", "danger", func() {
								go func() {
									apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/stop", iface.ID), nil, nil)
									refreshRoute()
								}()
							}),
						)
					}
					return Btn("Start", "primary", func() {
						go func() {
							apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/start", iface.ID), nil, nil)
							refreshRoute()
						}()
					})
				}(),
				Btn("Manage", "ghost", func() {
					navigate(fmt.Sprintf("/interfaces/%s", iface.ID))
				}),
			),
		),
	)
}
