//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Package-level state for edit modes (survives refreshRoute re-mount)
var (
	detailEditInterface bool
	detailEditPeerID    string
	detailSetKeyPeerID  string
	detailPollInterval  js.Value
)

func InterfaceDetailView(ifaceID string) loom.Node {
	status, setStatus := Signal[*interfaceStatusData](nil)
	loading, setLoading := Signal(true)

	loadStatus := func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", fmt.Sprintf("/api/v1/interfaces/%s/status", ifaceID), nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var s interfaceStatusData
			if err := json.Unmarshal(resp.Data, &s); err == nil {
				setStatus(&s)
			}
			setLoading(false)
		}()
	}

	Effect(func() {
		loadStatus()
		// Clear any previous poll interval
		if !detailPollInterval.IsUndefined() && !detailPollInterval.IsNull() {
			js.Global().Call("clearInterval", detailPollInterval)
		}
		// Poll for status updates every 5 seconds, skip during edits
		detailPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
			if !detailEditInterface && detailEditPeerID == "" && detailSetKeyPeerID == "" {
				loadStatus()
			}
			return nil
		}), 5000)
	})

	return Div(
		// Header with breadcrumb
		Div(
			Apply(Attr{"class": "flex items-center justify-between mb-6"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-3"}),
				Button(
					Apply(Attr{"class": "flex items-center gap-1 text-gray-400 hover:text-gray-700 text-sm transition-colors"}),
					Apply(On{"click": func() {
						detailEditInterface = false
						detailEditPeerID = ""
						detailSetKeyPeerID = ""
						navigate("/interfaces")
					}}),
					Icon("chevron-left", 16),
					Text("Interfaces"),
				),
				Span(Apply(Attr{"class": "text-gray-300"}), Text("/")),
				Span(Apply(Attr{"class": "font-mono text-lg font-semibold text-gray-900"}), Text(ifaceID)),
			),
			Div(
				Apply(Attr{"class": "flex gap-2"}),
				Btn("Add Peer", "primary", func() {
					detailEditInterface = false
					if detailEditPeerID == "__add__" {
						detailEditPeerID = ""
					} else {
						detailEditPeerID = "__add__"
					}
					refreshRoute()
				}),
				Btn("Delete", "danger", func() {
					ConfirmAction(fmt.Sprintf("Delete interface %s? This will remove all peers and cannot be undone.", ifaceID), func() {
						go func() {
							apiFetch("DELETE", fmt.Sprintf("/api/v1/interfaces/%s", ifaceID), nil, nil)
							navigate("/interfaces")
						}()
					})
				}),
			),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			s := status()
			if s == nil {
				return Div(EmptyState("Interface not found"))
			}

			// If in any edit mode, render static (no Bind) so form signals work
			if detailEditInterface || detailEditPeerID != "" || detailSetKeyPeerID != "" {
				return interfaceDetailContent(ifaceID, s, status)
			}

			// Read-only mode: wrap in Bind for live updates
			return Bind(func() loom.Node {
				s := status()
				if s == nil {
					return Div(EmptyState("Interface not found"))
				}
				return interfaceDetailContent(ifaceID, s, status)
			})
		}),
	)
}

func interfaceDetailContent(ifaceID string, s *interfaceStatusData, status Accessor[*interfaceStatusData]) loom.Node {
	return Div(
		// Status bar
		Div(
			Apply(Attr{"class": "flex items-center justify-between gap-3 mb-6 bg-white border border-gray-200 rounded-lg px-4 py-3"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2"}),
				func() loom.Node {
					if s.Running {
						return Fragment(
							Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 status-pulse"})),
							Span(Apply(Attr{"class": "text-sm text-emerald-600 font-medium"}), Text("Running")),
						)
					}
					return Fragment(
						Span(Apply(Attr{"class": "inline-block w-2.5 h-2.5 rounded-full bg-gray-300"})),
						Span(Apply(Attr{"class": "text-sm text-gray-400 font-medium"}), Text("Stopped")),
					)
				}(),
			),
			Div(
				Apply(Attr{"class": "flex flex-wrap gap-1.5"}),
				func() loom.Node {
					if s.Running {
						return Fragment(
							Btn("Restart", "ghost", func() {
								go func() {
									apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/restart", ifaceID), nil, nil)
									refreshRoute()
								}()
							}),
							Btn("Stop", "danger", func() {
								go func() {
									apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/stop", ifaceID), nil, nil)
									refreshRoute()
								}()
							}),
						)
					}
					return Btn("Start", "primary", func() {
						go func() {
							apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/start", ifaceID), nil, nil)
							refreshRoute()
						}()
					})
				}(),
				Btn("Sync", "ghost", func() {
					go func() {
						apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/sync", ifaceID), nil, nil)
						detailEditInterface = false
						detailEditPeerID = ""
						refreshRoute()
					}()
				}),
			),
		),

		// Interface settings card
		Card(
			Div(
				Apply(Attr{"class": "flex items-center justify-between mb-3"}),
				Span(Apply(Attr{"class": "text-xs text-gray-400 uppercase tracking-widest"}), Text("Interface Settings")),
				Btn(func() string {
					if detailEditInterface {
						return "Cancel"
					}
					return "Edit"
				}(), "ghost", func() {
					detailEditInterface = !detailEditInterface
					detailEditPeerID = ""
					refreshRoute()
				}),
			),
			func() loom.Node {
				if detailEditInterface {
					return interfaceEditForm(ifaceID, s.Interface)
				}
				return Div(
					Apply(Attr{"class": "grid grid-cols-2 sm:grid-cols-5 gap-3 sm:gap-4 text-sm"}),
					infoItem("Address", s.Interface.Address),
					infoItem("Port", fmt.Sprintf("%d", s.Interface.ListenPort)),
					infoItem("Endpoint", func() string {
						if s.Interface.Endpoint != "" {
							return s.Interface.Endpoint
						}
						return "(auto-detect)"
					}()),
					infoItem("Public Key", truncateKey(s.PublicKey)),
					infoItem("Available IPs", formatAvailableIPs(s.Interface.Address, len(s.Peers))),
				)
			}(),
		),

		// Add peer form
		func() loom.Node {
			if detailEditPeerID == "__add__" {
				// Collect used addresses for next-IP calculation
				var usedAddrs []string
				for _, ps := range s.Peers {
					if ps.Peer.Address != "" {
						usedAddrs = append(usedAddrs, ps.Peer.Address)
					}
				}
				return Div(
					Apply(Attr{"class": "mt-4"}),
					PeerForm(ifaceID, s.Interface.Address, usedAddrs, func() {
						detailEditPeerID = ""
						refreshRoute()
					}),
				)
			}
			return Span()
		}(),

		// Peers section
		Div(
			Apply(Attr{"class": "mt-6"}),
			CardHeader("Peers"),
			func() loom.Node {
				if len(s.Peers) == 0 {
					return EmptyState("No peers configured")
				}
				return peerTable(ifaceID, s.Peers)
			}(),
		),
	)
}

func interfaceEditForm(ifaceID string, iface interfaceData) loom.Node {
	address, setAddress := Signal(iface.Address)
	port, setPort := Signal(strconv.Itoa(iface.ListenPort))
	dns, setDNS := Signal(iface.DNS)
	mtu, setMTU := Signal(strconv.Itoa(iface.MTU))
	endpoint, setEndpoint := Signal(iface.Endpoint)
	errMsg, setErrMsg := Signal("")

	doSave := func() {
		setErrMsg("")
		portVal, err := strconv.Atoi(port())
		if err != nil || portVal < 1 || portVal > 65535 {
			setErrMsg("Port must be a number between 1 and 65535")
			return
		}
		mtuVal, err := strconv.Atoi(mtu())
		if err != nil || mtuVal < 1 {
			setErrMsg("MTU must be a positive number")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("PUT", fmt.Sprintf("/api/v1/interfaces/%s", ifaceID), map[string]any{
				"address":     address(),
				"listen_port": portVal,
				"dns":         dns(),
				"mtu":         mtuVal,
				"endpoint":    endpoint(),
			}, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			detailEditInterface = false
			refreshRoute()
		}()
	}

	return Div(
		ErrorAlert(errMsg),
		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Address (CIDR)", "text", "10.100.0.1/24", address, func(v string) { setAddress(v) }),
			FormField("Listen Port", "number", "51820", port, func(v string) { setPort(v) }),
			FormField("Public Endpoint", "text", "vpn.example.com", endpoint, func(v string) { setEndpoint(v) }),
			FormField("DNS", "text", "1.1.1.1", dns, func(v string) { setDNS(v) }),
			FormField("MTU", "number", "1420", mtu, func(v string) { setMTU(v) }),
		),
		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Save", "primary", doSave),
		),
	)
}

func infoItem(label, value string) loom.Node {
	return Div(
		Div(Apply(Attr{"class": "text-[11px] text-gray-400 uppercase tracking-widest mb-1"}), Text(label)),
		Div(Apply(Attr{"class": "font-mono text-gray-900"}), Text(value)),
	)
}

func formatAvailableIPs(address string, peerCount int) string {
	parts := strings.SplitN(address, "/", 2)
	if len(parts) != 2 {
		return "—"
	}
	prefix, err := strconv.Atoi(parts[1])
	if err != nil || prefix < 1 || prefix > 30 {
		return "—"
	}
	// Total usable host IPs: 2^(32-prefix) - 2 (network + broadcast) - 1 (gateway)
	total := (1 << (32 - prefix)) - 3
	if total < 0 {
		total = 0
	}
	available := total - peerCount
	if available < 0 {
		available = 0
	}
	return fmt.Sprintf("%d / %d", available, total)
}

func truncateKey(key string) string {
	if len(key) > 12 {
		return key[:8] + "..." + key[len(key)-4:]
	}
	return key
}

func peerSetKeyForm(ifaceID, peerID, peerName string) loom.Node {
	privKey, setPrivKey := Signal("")
	errMsg, setErrMsg := Signal("")
	success, setSuccess := Signal(false)

	doSet := func() {
		setErrMsg("")
		setSuccess(false)
		if privKey() == "" {
			setErrMsg("Private key is required")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("PUT", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/private-key", ifaceID, peerID), map[string]string{
				"private_key": privKey(),
			}, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			setSuccess(true)
			detailSetKeyPeerID = ""
			refreshRoute()
		}()
	}

	return Div(
		Div(
			Apply(Attr{"class": "text-sm font-medium text-gray-700 mb-3"}),
			Text(fmt.Sprintf("Set private key for %s", peerName)),
		),
		Div(
			Apply(Attr{"class": "text-xs text-gray-400 mb-3"}),
			Text("Paste the peer's WireGuard private key to enable config and QR code generation."),
		),
		ErrorAlert(errMsg),
		Bind(func() loom.Node {
			if success() {
				return Div(
					Apply(Attr{"class": "mb-3 p-3 bg-emerald-50 border border-emerald-200 rounded-md text-emerald-700 text-sm"}),
					Text("Private key set successfully"),
				)
			}
			return Div(Apply(Attr{"class": "hidden"}), Text(""))
		}),
		Div(
			Apply(Attr{"class": "flex items-end gap-3"}),
			Div(
				Apply(Attr{"class": "flex-1"}),
				FormField("Private Key", "password", "base64-encoded WireGuard private key", privKey, func(v string) { setPrivKey(v) }),
			),
			Div(
				Apply(Attr{"class": "flex gap-2 pb-4"}),
				Btn("Save Key", "primary", doSet),
				Btn("Cancel", "ghost", func() {
					detailSetKeyPeerID = ""
					refreshRoute()
				}),
			),
		),
	)
}

func peerActions(ifaceID string, ps peerStatusData) loom.Node {
	return Div(
		Apply(Attr{"class": "flex items-center gap-0.5"}),
		func() loom.Node {
			if ps.HasPrivateKey {
				return IconBtn("qr-code", "View config", func() {
					navigate(fmt.Sprintf("/interfaces/%s/peers/%s/config", ifaceID, ps.Peer.ID))
				})
			}
			return IconBtn("key-round", "Set private key", func() {
				detailEditInterface = false
				detailEditPeerID = ""
				if detailSetKeyPeerID == ps.Peer.ID {
					detailSetKeyPeerID = ""
				} else {
					detailSetKeyPeerID = ps.Peer.ID
				}
				refreshRoute()
			})
		}(),
		IconBtn("pencil", "Edit peer", func() {
			detailEditInterface = false
			detailEditPeerID = ps.Peer.ID
			refreshRoute()
		}),
		func() loom.Node {
			if ps.Peer.Enabled {
				return IconBtn("eye-off", "Disable peer", func() {
					ConfirmAction(fmt.Sprintf("Disable peer %s? The peer will be disconnected immediately.", ps.Peer.Name), func() {
						go func() {
							apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/disable", ifaceID, ps.Peer.ID), nil, nil)
							detailEditPeerID = ""
							refreshRoute()
						}()
					})
				})
			}
			return IconBtn("eye", "Enable peer", func() {
				go func() {
					apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/enable", ifaceID, ps.Peer.ID), nil, nil)
					detailEditPeerID = ""
					refreshRoute()
				}()
			})
		}(),
		IconBtnDanger("trash-2", "Remove peer", func() {
			ConfirmAction(fmt.Sprintf("Remove peer %s? This cannot be undone.", ps.Peer.Name), func() {
				go func() {
					apiFetch("DELETE", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s", ifaceID, ps.Peer.ID), nil, nil)
					detailEditPeerID = ""
					refreshRoute()
				}()
			})
		}),
	)
}

func peerTable(ifaceID string, peers []peerStatusData) loom.Node {
	cards := make([]loom.Node, 0, len(peers))
	rows := make([]loom.Node, 0, len(peers)*2)

	for _, ps := range peers {
		ps := ps

		// Mobile card
		cardChildren := []loom.Node{
			Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg px-4 py-3"}),
			// Header: status dot + name + actions
			Div(
				Apply(Attr{"class": "flex items-center justify-between"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-2 min-w-0"}),
					StatusDot(ps.Peer.Enabled, ps.Connected),
					Span(Apply(Attr{"class": "text-sm font-medium text-gray-900 truncate"}), Text(ps.Peer.Name)),
				),
				peerActions(ifaceID, ps),
			),
			// Details
			Div(
				Apply(Attr{"class": "flex items-center justify-between mt-1.5 text-xs font-mono text-gray-400 pl-[18px]"}),
				Span(Text(ps.Peer.Address)),
				Span(Text(fmt.Sprintf("↓%s ↑%s", FormatBytes(ps.TransferRx), FormatBytes(ps.TransferTx)))),
			),
		}
		// Inline edit form below card
		if detailEditPeerID == ps.Peer.ID {
			cardChildren = append(cardChildren, Div(
				Apply(Attr{"class": "mt-3 pt-3 border-t border-gray-100"}),
				PeerEditForm(ifaceID, ps.Peer, func() {
					detailEditPeerID = ""
					refreshRoute()
				}),
			))
		}
		if detailSetKeyPeerID == ps.Peer.ID {
			cardChildren = append(cardChildren, Div(
				Apply(Attr{"class": "mt-3 pt-3 border-t border-gray-100"}),
				peerSetKeyForm(ifaceID, ps.Peer.ID, ps.Peer.Name),
			))
		}
		cards = append(cards, Div(cardChildren...))

		// Desktop table row
		rows = append(rows, Elem("tr",
			Apply(Attr{"class": "border-b border-gray-100 hover:bg-gray-50 transition-colors"}),
			Elem("td", Apply(Attr{"class": "px-4 py-3"}), StatusDot(ps.Peer.Enabled, ps.Connected)),
			Elem("td", Apply(Attr{"class": "px-4 py-3"}),
				Div(
					Div(Apply(Attr{"class": "text-sm font-medium text-gray-900"}), Text(ps.Peer.Name)),
					Div(Apply(Attr{"class": "font-mono text-[11px] text-gray-400"}), Text(truncateKey(ps.Peer.PublicKey))),
				),
			),
			Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-xs text-gray-500"}), Text(ps.Peer.Address)),
			Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-xs text-gray-500"}),
				func() loom.Node {
					parts := strings.Split(ps.Peer.AllowedIPs, ",")
					nodes := make([]loom.Node, 0, len(parts))
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if p != "" {
							nodes = append(nodes, Div(Text(p)))
						}
					}
					return Div(nodes...)
				}(),
			),
			Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-xs text-gray-500"}),
				func() loom.Node {
					if ps.Peer.ClientAllowedIPs == "" {
						return Span(Apply(Attr{"class": "text-gray-300"}), Text("0.0.0.0/0"))
					}
					parts := strings.Split(ps.Peer.ClientAllowedIPs, ",")
					nodes := make([]loom.Node, 0, len(parts))
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if p != "" {
							nodes = append(nodes, Div(Text(p)))
						}
					}
					return Div(nodes...)
				}(),
			),
			Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-xs text-gray-400"}), Text(FormatBytes(ps.TransferRx))),
			Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-xs text-gray-400"}), Text(FormatBytes(ps.TransferTx))),
			Elem("td", Apply(Attr{"class": "px-4 py-3"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-0.5 justify-end"}),
					peerActions(ifaceID, ps),
				),
			),
		))
		// Inline edit form row (desktop)
		if detailEditPeerID == ps.Peer.ID {
			rows = append(rows, Elem("tr",
				Apply(Attr{"class": "border-b border-gray-100 bg-gray-50"}),
				Elem("td", Apply(Attr{"class": "p-4", "colspan": "8"}),
					PeerEditForm(ifaceID, ps.Peer, func() {
						detailEditPeerID = ""
						refreshRoute()
					}),
				),
			))
		}
		// Inline set private key form row (desktop)
		if detailSetKeyPeerID == ps.Peer.ID {
			rows = append(rows, Elem("tr",
				Apply(Attr{"class": "border-b border-gray-100 bg-gray-50"}),
				Elem("td", Apply(Attr{"class": "p-4", "colspan": "8"}),
					peerSetKeyForm(ifaceID, ps.Peer.ID, ps.Peer.Name),
				),
			))
		}
	}

	return Div(
		// Mobile cards
		Div(
			Apply(Attr{"class": "md:hidden space-y-3"}),
			Fragment(cards...),
		),
		// Desktop table
		Div(
			Apply(Attr{"class": "hidden md:block bg-white border border-gray-200 rounded-lg overflow-hidden"}),
			Elem("table",
				Apply(Attr{"class": "w-full text-sm"}),
				Elem("thead",
					Elem("tr",
						Apply(Attr{"class": "border-b border-gray-200 text-left text-[11px] uppercase tracking-widest text-gray-400"}),
						Elem("th", Apply(Attr{"class": "px-4 py-3 w-8"})),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Name")),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Address")),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Server IPs")),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Client IPs")),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("RX")),
						Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("TX")),
						Elem("th", Apply(Attr{"class": "px-4 py-3 text-right"}), Text("Actions")),
					),
				),
				Elem("tbody", rows...),
			),
		),
	)
}
