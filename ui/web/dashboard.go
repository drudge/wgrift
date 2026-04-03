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

				// Health indicator dot
				healthDot := "w-3 h-3 rounded-full flex-shrink-0 "
				if runningCount > 0 && runningCount == len(d.Interfaces) {
					healthDot += "bg-green-400 status-pulse"
				} else if runningCount > 0 {
					healthDot += "bg-amber-400"
				} else {
					healthDot += "bg-ink-4"
				}

				// Running count color — green when all up, muted when none
				countColor := "text-ink-2"
				if runningCount > 0 && runningCount == len(d.Interfaces) {
					countColor = "text-green-400"
				} else if runningCount > 0 {
					countColor = "text-ink-1"
				}

				ifaceNodes := make([]loom.Node, 0)
				if len(d.Interfaces) > 0 {
					ifaceNodes = interfaceCards(d.Interfaces)
				}

				return Div(
					// ── Page header ──
					Div(
						Apply(Attr{"class": "mb-8"}),
						// Title row with health dot
						Div(
							Apply(Attr{"class": "flex items-center gap-3 mb-1.5"}),
							Div(Apply(Attr{"class": healthDot})),
							H2(
								Apply(Attr{"class": "text-2xl font-bold tracking-tight"}),
								Span(Apply(Attr{"class": countColor + " font-mono tabular-nums"}),
									Text(fmt.Sprintf("%d", runningCount))),
								Span(Apply(Attr{"class": "text-ink-1"}),
									Text(fmt.Sprintf(" of %d %s running", len(d.Interfaces), pluralize(len(d.Interfaces), "interface", "interfaces")))),
							),
						),
						// Stats subtitle
						Div(
							Apply(Attr{"class": "text-sm text-ink-2 font-mono tabular-nums pl-6"}),
							Span(Text(fmt.Sprintf("%d peers · %d active", d.TotalPeers, d.ActivePeers))),
							Span(Apply(Attr{"class": "hidden sm:inline"}), Text(fmt.Sprintf(" · ↓ %s · ↑ %s", FormatBytes(d.TotalRx), FormatBytes(d.TotalTx)))),
							Div(Apply(Attr{"class": "sm:hidden text-xs text-ink-3 mt-0.5"}), Text(fmt.Sprintf("↓ %s · ↑ %s", FormatBytes(d.TotalRx), FormatBytes(d.TotalTx)))),
						),
					),

					// ── Active Connections ──
					activeConnectionsSection(d.ActiveConnections),

					// ── Interfaces ──
					Div(
						Div(
							Apply(Attr{"class": "flex items-center justify-between mb-4"}),
							H3(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text("Interfaces")),
							Btn("New Interface", "primary", func() { navigate("/interfaces?action=create") }),
						),
						func() loom.Node {
							if len(ifaceNodes) == 0 {
								return EmptyState("No interfaces configured")
							}
							return Div(
								Apply(Attr{"class": "space-y-2"}),
								Fragment(ifaceNodes...),
							)
						}(),
					),
				)
			})
		}),
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
	dotClass := "w-2 h-2 rounded-full bg-ink-4 flex-shrink-0"
	statusText := "Stopped"
	statusClass := "text-[11px] text-ink-4 font-medium"
	if iface.Running {
		dotClass = "w-2 h-2 rounded-full bg-green-400 flex-shrink-0"
		statusText = "Running"
		statusClass = "text-[11px] text-green-400/70 font-medium"
	}

	// Stats line: peers + optional traffic
	statsNodes := []loom.Node{
		Span(
			Span(Apply(Attr{"class": "text-ink-2 font-semibold"}), Text(fmt.Sprintf("%d", iface.ConnectedPeers))),
			Text(fmt.Sprintf("/%d peers", iface.TotalPeers)),
		),
	}
	if iface.TotalRx > 0 || iface.TotalTx > 0 {
		statsNodes = append(statsNodes, Span(
			Apply(Attr{"class": "text-ink-3"}),
			Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(iface.TotalRx), FormatBytes(iface.TotalTx))),
		))
	}

	// Action buttons
	actionBtns := func() loom.Node {
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
	}()

	return Div(
		Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg hover:border-line-3 transition-all duration-150"}),
		Div(
			Apply(Attr{"class": "px-5 py-4"}),

			// Desktop: two-column layout — info left, buttons right
			Div(
				Apply(Attr{"class": "hidden sm:flex items-center justify-between gap-4"}),
				// Left: all info stacked
				Div(
					Apply(Attr{"class": "flex items-start gap-3 min-w-0"}),
					Div(Apply(Attr{"class": dotClass + " mt-1.5"})),
					Div(
						Apply(Attr{"class": "min-w-0"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2.5"}),
							Span(Apply(Attr{"class": "font-mono text-sm font-bold text-ink-1"}), Text(iface.ID)),
							Span(Apply(Attr{"class": statusClass}), Text(statusText)),
						),
						Div(
							Apply(Attr{"class": "font-mono text-xs text-ink-3 mt-0.5"}),
							Text(fmt.Sprintf("%s · :%d", iface.Address, iface.ListenPort)),
						),
						Div(
							append([]loom.Node{Apply(Attr{"class": "flex items-center gap-4 text-xs font-mono text-ink-2 mt-1.5"})}, statsNodes...)...,
						),
					),
				),
				// Right: buttons vertically centered
				Div(
					Apply(Attr{"class": "flex items-center gap-1 flex-shrink-0"}),
					actionBtns,
					Btn("Manage →", "ghost", func() {
						navigate(fmt.Sprintf("/interfaces/%s", iface.ID))
					}),
				),
			),

			// Mobile: stacked layout
			Div(
				Apply(Attr{"class": "sm:hidden"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-3 min-w-0"}),
					Div(Apply(Attr{"class": dotClass})),
					Div(
						Apply(Attr{"class": "min-w-0"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2.5"}),
							Span(Apply(Attr{"class": "font-mono text-sm font-bold text-ink-1"}), Text(iface.ID)),
							Span(Apply(Attr{"class": statusClass}), Text(statusText)),
						),
						Div(
							Apply(Attr{"class": "font-mono text-xs text-ink-3 mt-0.5"}),
							Text(fmt.Sprintf("%s · :%d", iface.Address, iface.ListenPort)),
						),
					),
				),
				Div(
					append([]loom.Node{Apply(Attr{"class": "flex items-center gap-3 text-xs font-mono text-ink-3 mt-2 pl-5"})}, statsNodes...)...,
				),
				Div(
					Apply(Attr{"class": "flex items-center gap-1 mt-3 pl-5 border-t border-line-1 pt-3"}),
					actionBtns,
					Btn("Manage →", "ghost", func() {
						navigate(fmt.Sprintf("/interfaces/%s", iface.ID))
					}),
				),
			),
		),
	)
}

func activeConnectionsSection(connections []activeConnectionData) loom.Node {
	return Div(
		Apply(Attr{"class": "mb-8"}),
		Div(
			Apply(Attr{"class": "flex items-center justify-between mb-4"}),
			H3(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text("Active Connections")),
			Span(Apply(Attr{"class": "text-xs font-mono text-ink-4"}), Text(fmt.Sprintf("%d connected", len(connections)))),
		),
		func() loom.Node {
			if len(connections) == 0 {
				return EmptyState("No active connections")
			}
			rows := make([]loom.Node, 0, len(connections))
			for _, conn := range connections {
				conn := conn
				rows = append(rows, activeConnectionRow(conn))
			}
			return Div(
				Apply(Attr{"class": "space-y-2"}),
				Fragment(rows...),
			)
		}(),
	)
}

func activeConnectionRow(conn activeConnectionData) loom.Node {
	clickNav := func() { navigate(fmt.Sprintf("/interfaces/%s", conn.InterfaceID)) }

	return Div(
		Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg hover:border-line-3 transition-all duration-150 cursor-pointer"}),
		Apply(On{"click": clickNav}),
		Div(
			Apply(Attr{"class": "px-5 py-3.5"}),

			// Desktop
			Div(
				Apply(Attr{"class": "hidden sm:flex items-center justify-between gap-4"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-3 min-w-0"}),
					Div(Apply(Attr{"class": "w-2 h-2 rounded-full bg-green-400 flex-shrink-0"})),
					Div(
						Apply(Attr{"class": "min-w-0"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2.5"}),
							Span(Apply(Attr{"class": "text-sm font-semibold text-ink-1 truncate"}), Text(conn.PeerName)),
							Badge(conn.InterfaceID, ""),
							PeerTypeBadge(conn.PeerType),
						),
						Div(
							Apply(Attr{"class": "flex items-center gap-4 font-mono text-xs text-ink-3 mt-0.5"}),
							Span(Apply(Attr{"class": "text-ink-2"}), Text(conn.Address)),
							func() loom.Node {
								if conn.Endpoint != "" {
									return Span(
										Span(Apply(Attr{"class": "text-ink-4"}), Text("from ")),
										Text(conn.Endpoint),
									)
								}
								return Span()
							}(),
							Span(Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(conn.TransferRx), FormatBytes(conn.TransferTx)))),
						),
					),
				),
				Span(
					Apply(Attr{"class": "text-ink-3 text-sm flex-shrink-0"}),
					Text("→"),
				),
			),

			// Mobile
			Div(
				Apply(Attr{"class": "sm:hidden"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-3 min-w-0"}),
					Div(Apply(Attr{"class": "w-2 h-2 rounded-full bg-green-400 flex-shrink-0"})),
					Div(
						Apply(Attr{"class": "min-w-0 flex-1"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2"}),
							Span(Apply(Attr{"class": "text-sm font-semibold text-ink-1 truncate"}), Text(conn.PeerName)),
							Badge(conn.InterfaceID, ""),
						),
						Div(
							Apply(Attr{"class": "font-mono text-xs text-ink-3 mt-0.5"}),
							Text(conn.Address),
						),
					),
				),
				Div(
					Apply(Attr{"class": "flex items-center gap-3 text-xs font-mono text-ink-3 mt-2 pl-5"}),
					func() loom.Node {
						if conn.Endpoint != "" {
							return Span(Text(conn.Endpoint))
						}
						return Span()
					}(),
					Span(Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(conn.TransferRx), FormatBytes(conn.TransferTx)))),
				),
			),
		),
	)
}
