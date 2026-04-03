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
	detailCurrentID     string
	detailEditInterface bool
	detailEditPeerID    string
	detailSetKeyPeerID  string
	detailPollInterval  js.Value
)

func InterfaceDetailView(ifaceID string) loom.Node {
	// Reset edit state only when navigating to a different interface
	if ifaceID != detailCurrentID {
		detailEditInterface = false
		detailEditPeerID = ""
		detailSetKeyPeerID = ""
		detailCurrentID = ifaceID
	}

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
		// Breadcrumb row — everything on one line on desktop, wraps on mobile
		Div(
			Apply(Attr{"class": "flex flex-wrap items-center justify-between gap-3 mb-6"}),
			// Left: breadcrumb
			Div(
				Apply(Attr{"class": "flex items-center gap-2"}),
				Button(
					Apply(Attr{"class": "flex items-center gap-1 text-ink-3 hover:text-wg-400 text-sm transition-colors"}),
					Apply(On{"click": func() {
						detailEditInterface = false
						detailEditPeerID = ""
						detailSetKeyPeerID = ""
						navigate("/interfaces")
					}}),
					Icon("chevron-left", 16),
					Text("Interfaces"),
				),
				Span(Apply(Attr{"class": "text-ink-4/40"}), Text("/")),
				Span(Apply(Attr{"class": "font-mono text-lg font-bold text-ink-1 tracking-tight"}), Text(ifaceID)),
			),
			// Right: actions — wrap on mobile
			Div(
				Apply(Attr{"class": "flex items-center gap-2"}),
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
	// Status bar stripe and indicator classes based on running state
	stripeClass := "h-[2px] bg-ink-4/30"
	dotClass := "inline-block w-2.5 h-2.5 rounded-full bg-ink-4"
	labelClass := "text-sm text-ink-3 font-medium"
	labelText := "Stopped"
	if s.Running {
		stripeClass = "h-[2px] bg-green-500"
		dotClass = "inline-block w-2.5 h-2.5 rounded-full bg-green-500 status-pulse"
		labelClass = "text-sm text-green-400 font-medium"
		labelText = "Running"
	}

	return Div(
		// Status bar
		Div(
			Apply(Attr{"class": "bg-surface-1 rounded-lg overflow-hidden mb-6 border border-line-1"}),
			// Gradient stripe at top
			Div(Apply(Attr{"class": stripeClass})),
			Div(
				Apply(Attr{"class": "px-7 py-5 flex items-center justify-between"}),
				// Left: status indicator + text
				Div(
					Apply(Attr{"class": "flex items-center gap-3"}),
					Span(Apply(Attr{"class": dotClass})),
					Span(Apply(Attr{"class": labelClass}), Text(labelText)),
				),
				// Right: action buttons
				Div(
					Apply(Attr{"class": "flex flex-wrap items-center gap-2"}),
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
		),

		// Interface settings card
		Div(
			Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg p-6"}),
			Div(
				Apply(Attr{"class": "flex items-center justify-between mb-4"}),
				Span(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text("Interface Settings")),
				func() loom.Node {
					if !detailEditInterface {
						return Btn("Edit", "ghost", func() {
							detailEditInterface = true
							detailEditPeerID = ""
							refreshRoute()
						})
					}
					return Span()
				}(),
			),
			func() loom.Node {
				if detailEditInterface {
					return interfaceEditForm(ifaceID, s.Interface, s.PublicKey)
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
					copyableInfoItem("Public Key", s.PublicKey),
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
					Apply(Attr{"class": "mt-6"}),
					PeerForm(ifaceID, s.Interface.Address, usedAddrs, func() {
						detailEditPeerID = ""
						refreshRoute()
					}, func() {
						detailEditPeerID = ""
						refreshRoute()
					}),
				)
			}
			return Span()
		}(),

		// Peers section
		Div(
			Apply(Attr{"class": "mt-8"}),
			Div(
				Apply(Attr{"class": "flex items-center justify-between mb-5"}),
				H3(Apply(Attr{"class": "text-xs font-semibold text-ink-4 uppercase tracking-[0.2em]"}), Text("Peers")),
				Span(Apply(Attr{"class": "text-xs font-mono text-ink-4"}), Text(fmt.Sprintf("%d total", len(s.Peers)))),
			),
			func() loom.Node {
				if len(s.Peers) == 0 {
					return EmptyState("No peers configured")
				}
				return peerCardList(ifaceID, s.Peers)
			}(),
		),
	)
}

func interfaceEditForm(ifaceID string, iface interfaceData, publicKey string) loom.Node {
	address, setAddress := Signal(iface.Address)
	port, setPort := Signal(strconv.Itoa(iface.ListenPort))
	dns, setDNS := Signal(iface.DNS)
	mtu, setMTU := Signal(strconv.Itoa(iface.MTU))
	endpoint, setEndpoint := Signal(iface.Endpoint)
	errMsg, setErrMsg := Signal(ErrorInfo{})
	FocusInput(`input[placeholder="10.100.0.1/24"]`)

	doSave := func() {
		setErrMsg(ErrorInfo{})
		portVal, err := strconv.Atoi(port())
		if err != nil || portVal < 1 || portVal > 65535 {
			setErrMsg(ErrorInfo{Message: "Port must be a number between 1 and 65535"})
			return
		}
		mtuVal, err := strconv.Atoi(mtu())
		if err != nil || mtuVal < 1 {
			setErrMsg(ErrorInfo{Message: "MTU must be a positive number"})
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
				setErrMsg(apiErrorInfo(err))
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
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
			readOnlyCopyField("Public Key", publicKey),
		),
		Div(
			Apply(Attr{"class": "flex items-center gap-2 mt-2"}),
			Btn("Save", "primary", doSave),
			Btn("Cancel", "ghost", func() {
				detailEditInterface = false
				refreshRoute()
			}),
		),
	)
}

func infoItem(label, value string) loom.Node {
	return Div(
		Div(Apply(Attr{"class": "text-[11px] text-ink-3 uppercase tracking-widest font-medium mb-1.5"}), Text(label)),
		Div(Apply(Attr{"class": "font-mono text-ink-1 text-sm"}), Text(value)),
	)
}

func formatAvailableIPs(address string, peerCount int) string {
	parts := strings.SplitN(address, "/", 2)
	if len(parts) != 2 {
		return "\u2014"
	}
	prefix, err := strconv.Atoi(parts[1])
	if err != nil || prefix < 1 || prefix > 30 {
		return "\u2014"
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

// copyableKey renders a clickable public key that copies to clipboard with a toast.
// Shows truncated on mobile, full key on desktop.
func copyableKey(key string) loom.Node {
	return Div(
		Apply(Attr{
			"class": "font-mono text-[11px] text-ink-4 truncate cursor-pointer hover:text-ink-2 transition-colors",
			"title": "Click to copy",
		}),
		Apply(On{"click": func() {
			js.Global().Get("navigator").Get("clipboard").Call("writeText", key)
			showToast("Public key copied")
		}}),
		Span(Apply(Attr{"class": "sm:hidden"}), Text(truncateKey(key))),
		Span(Apply(Attr{"class": "hidden sm:inline"}), Text(key)),
	)
}

// copyableInfoItem renders a label+value info item where the value is clickable to copy.
func copyableInfoItem(label, value string) loom.Node {
	return Div(
		Div(Apply(Attr{"class": "text-[11px] text-ink-3 uppercase tracking-widest font-medium mb-1.5"}), Text(label)),
		Div(
			Apply(Attr{
				"class": "font-mono text-ink-1 text-sm cursor-pointer hover:text-wg-500 transition-colors truncate",
				"title": "Click to copy",
			}),
			Apply(On{"click": func() {
				js.Global().Get("navigator").Get("clipboard").Call("writeText", value)
				showToast("Public key copied")
			}}),
			Text(truncateKey(value)),
		),
	)
}

// readOnlyCopyField renders a read-only form field with click-to-copy.
func readOnlyCopyField(label, value string) loom.Node {
	return Div(
		Apply(Attr{"class": "mb-4"}),
		Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text(label)),
		Div(
			Apply(Attr{
				"class": "w-full px-3.5 py-2.5 bg-surface-0/50 border border-line-1 rounded-md text-ink-3 text-sm font-mono truncate cursor-pointer hover:text-ink-1 hover:border-line-2 transition-colors",
				"title": "Click to copy",
			}),
			Apply(On{"click": func() {
				js.Global().Get("navigator").Get("clipboard").Call("writeText", value)
				showToast("Public key copied")
			}}),
			Text(value),
		),
	)
}

func peerSetKeyForm(ifaceID, peerID, peerName string) loom.Node {
	privKey, setPrivKey := Signal("")
	errMsg, setErrMsg := Signal(ErrorInfo{})
	success, setSuccess := Signal(false)
	FocusInput(`input[placeholder="base64-encoded WireGuard private key"]`)

	doSet := func() {
		setErrMsg(ErrorInfo{})
		setSuccess(false)
		if privKey() == "" {
			setErrMsg(ErrorInfo{Message: "Private key is required"})
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("PUT", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/private-key", ifaceID, peerID), map[string]string{
				"private_key": privKey(),
			}, &resp)
			if err != nil {
				setErrMsg(apiErrorInfo(err))
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
				return
			}
			setSuccess(true)
			detailSetKeyPeerID = ""
			refreshRoute()
		}()
	}

	return Div(
		Div(
			Apply(Attr{"class": "text-sm font-medium text-ink-1 mb-3"}),
			Text(fmt.Sprintf("Set private key for %s", peerName)),
		),
		Div(
			Apply(Attr{"class": "text-xs text-ink-3 mb-3"}),
			Text("Paste the peer's WireGuard private key to enable config and QR code generation."),
		),
		ErrorAlert(errMsg),
		Bind(func() loom.Node {
			if success() {
				return Div(
					Apply(Attr{"class": "mb-3 p-3 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm"}),
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
				return IconBtn("power-off", "Disable peer", func() {
					ConfirmAction(fmt.Sprintf("Disable peer %s? The peer will be disconnected immediately.", ps.Peer.Name), func() {
						go func() {
							apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/disable", ifaceID, ps.Peer.ID), nil, nil)
							detailEditPeerID = ""
							refreshRoute()
						}()
					})
				})
			}
			return IconBtn("power", "Enable peer", func() {
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

// parseCSV splits a comma-separated string into trimmed entries.
func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// peerCardList renders all peers as card-based rows (replaces the old table+mobile-cards split).
func peerCardList(ifaceID string, peers []peerStatusData) loom.Node {
	cards := make([]loom.Node, 0, len(peers))

	for _, ps := range peers {
		ps := ps

		cardChildren := []loom.Node{
			Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg px-6 py-4 hover:bg-surface-2/60 transition-colors"}),
			// Top row: status + name/key + actions
			Div(
				Apply(Attr{"class": "flex items-center justify-between"}),
				Div(
					Apply(Attr{"class": "flex items-center gap-3 min-w-0"}),
					StatusDot(ps.Peer.Enabled, ps.Connected),
					Div(
						Apply(Attr{"class": "min-w-0"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-2"}),
							Span(Apply(Attr{"class": "text-sm font-semibold text-ink-1 truncate"}), Text(ps.Peer.Name)),
							PeerTypeBadge(ps.Peer.Type),
						),
						copyableKey(ps.Peer.PublicKey),
					),
				),
				peerActions(ifaceID, ps),
			),
			// Bottom row: address + endpoint + transfer
			Div(
				Apply(Attr{"class": "flex flex-wrap items-center gap-x-5 gap-y-1 mt-3 pl-[22px] text-xs font-mono text-ink-3"}),
				Span(Apply(Attr{"class": "text-ink-2"}), Text(ps.Peer.Address)),
				func() loom.Node {
					if ps.Connected && ps.Endpoint != "" {
						return Span(
							Span(Apply(Attr{"class": "text-ink-4"}), Text("from ")),
							Text(ps.Endpoint),
						)
					}
					return Span()
				}(),
				Span(Text(fmt.Sprintf("↓%s  ↑%s", FormatBytes(ps.TransferRx), FormatBytes(ps.TransferTx)))),
				func() loom.Node {
					if ips := parseCSV(ps.Peer.AllowedIPs); len(ips) > 0 {
						n := len(ips)
						label := fmt.Sprintf("%d Server IPs", n)
						if n == 1 {
							label = "1 Server IP"
						}
						return Tooltip(Badge(label, ""), ips)
					}
					return Span()
				}(),
				func() loom.Node {
					if ips := parseCSV(ps.Peer.ClientAllowedIPs); len(ips) > 0 {
						n := len(ips)
						label := fmt.Sprintf("%d Client IPs", n)
						if n == 1 {
							label = "1 Client IP"
						}
						return Tooltip(Badge(label, ""), ips)
					}
					return Span()
				}(),
				func() loom.Node {
					if dns := parseCSV(ps.Peer.DNS); len(dns) > 0 {
						return Tooltip(Badge("DNS", ""), dns)
					}
					return Span()
				}(),
			),
		}

		// Inline edit form below card content
		if detailEditPeerID == ps.Peer.ID {
			cardChildren = append(cardChildren, Div(
				Apply(Attr{"class": "mt-4 pt-4 border-t border-line-1"}),
				PeerEditForm(ifaceID, ps.Peer, func() {
					detailEditPeerID = ""
					refreshRoute()
				}),
			))
		}
		// Inline set private key form below card content
		if detailSetKeyPeerID == ps.Peer.ID {
			cardChildren = append(cardChildren, Div(
				Apply(Attr{"class": "mt-4 pt-4 border-t border-line-1"}),
				peerSetKeyForm(ifaceID, ps.Peer.ID, ps.Peer.Name),
			))
		}

		cards = append(cards, Div(cardChildren...))
	}

	return Div(
		Apply(Attr{"class": "space-y-3"}),
		Fragment(cards...),
	)
}
