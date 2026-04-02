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

// Package-level state for form toggle (survives refreshRoute re-mount)
var interfacesFormMode = "none" // "none", "create", "import"

func InterfacesView() loom.Node {
	ifaces, setIfaces := Signal[[]interfaceData](nil)
	loading, setLoading := Signal(true)

	// Check URL params on first mount
	search := js.Global().Get("window").Get("location").Get("search").String()
	if strings.Contains(search, "action=create") {
		interfacesFormMode = "create"
	} else if strings.Contains(search, "action=import") {
		interfacesFormMode = "import"
	}

	loadIfaces := func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/interfaces", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []interfaceData
			json.Unmarshal(resp.Data, &list)
			setIfaces(list)
			setLoading(false)
		}()
	}

	Effect(func() { loadIfaces() })

	return Div(
		PageHeader("Interfaces", "Manage WireGuard tunnel interfaces",
			Btn("Import Config", "ghost", func() {
				if interfacesFormMode == "import" {
					interfacesFormMode = "none"
				} else {
					interfacesFormMode = "import"
				}
				// Clear URL params so they don't re-open the form on refresh
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/interfaces")
				refreshRoute()
			}),
			Btn("New Interface", "primary", func() {
				if interfacesFormMode == "create" {
					interfacesFormMode = "none"
				} else {
					interfacesFormMode = "create"
				}
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/interfaces")
				refreshRoute()
			}),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			// Form area — rendered after interfaces are loaded so defaults work
			formNode := func() loom.Node {
				switch interfacesFormMode {
				case "create":
					return Div(
						Apply(Attr{"class": "mb-6"}),
						createInterfaceForm(ifaces(), func() {
							interfacesFormMode = "none"
							js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/interfaces")
							refreshRoute()
						}),
					)
				case "import":
					return Div(
						Apply(Attr{"class": "mb-6"}),
						importInterfaceForm(func() {
							interfacesFormMode = "none"
							js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/interfaces")
							refreshRoute()
						}),
					)
				default:
					return Span()
				}
			}()

			list := ifaces()
			if len(list) == 0 {
				return Div(formNode, EmptyState("No interfaces configured"))
			}

			// Mobile cards
			cards := make([]loom.Node, 0, len(list))
			// Desktop card rows
			rows := make([]loom.Node, 0, len(list))
			for _, iface := range list {
				iface := iface
				clickNav := func() { navigate(fmt.Sprintf("/interfaces/%s", iface.ID)) }
				badge := func() loom.Node {
					if iface.Enabled {
						return Badge("enabled", "emerald")
					}
					return Badge("disabled", "")
				}()

				// Mobile card
				cards = append(cards, Div(
					Apply(Attr{"class": "bg-surface-1 border border-line-1 rounded-lg px-5 py-4 active:bg-surface-2"}),
					Apply(On{"click": clickNav}),
					Div(
						Apply(Attr{"class": "flex items-center justify-between mb-2"}),
						Span(Apply(Attr{"class": "font-mono text-sm font-medium text-ink-1"}), Text(iface.ID)),
						badge,
					),
					Div(
						Apply(Attr{"class": "font-mono text-xs text-ink-3"}),
						Text(fmt.Sprintf("%s · port %d", iface.Address, iface.ListenPort)),
					),
				))

				// Desktop card row
				rows = append(rows, Div(
					Apply(Attr{"class": "bg-surface-1 rounded-lg px-6 py-4 flex items-center justify-between hover:bg-surface-2/60 transition-colors cursor-pointer"}),
					Apply(On{"click": clickNav}),
					// Left: primary info
					Div(
						Apply(Attr{"class": "flex items-center gap-5 min-w-0"}),
						Span(Apply(Attr{"class": "font-mono text-sm font-bold text-ink-1 w-24 flex-shrink-0"}), Text(iface.ID)),
						Span(Apply(Attr{"class": "font-mono text-sm text-ink-3"}), Text(fmt.Sprintf("%s · :%d", iface.Address, iface.ListenPort))),
					),
					// Right: badge
					badge,
				))
			}

			return Div(
				formNode,
				// Mobile cards
				Div(
					Apply(Attr{"class": "md:hidden space-y-3"}),
					Fragment(cards...),
				),
				// Desktop card rows
				Div(
					Apply(Attr{"class": "hidden md:block space-y-2"}),
					Fragment(rows...),
				),
			)
		}),
	)
}

func nextInterfaceDefaults(existing []interfaceData) (string, string, string) {
	usedIDs := make(map[string]bool)
	usedPorts := make(map[int]bool)
	for _, iface := range existing {
		usedIDs[iface.ID] = true
		usedPorts[iface.ListenPort] = true
	}
	// Next wgN
	nextID := "wg0"
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("wg%d", i)
		if !usedIDs[candidate] {
			nextID = candidate
			break
		}
	}
	// Next port starting from 51820
	nextPort := 51820
	for usedPorts[nextPort] {
		nextPort++
	}
	// Next subnet 10.100.N.1/24
	nextAddr := fmt.Sprintf("10.100.%d.1/24", len(existing))
	return nextID, strconv.Itoa(nextPort), nextAddr
}

func createInterfaceForm(existing []interfaceData, onCreated func()) loom.Node {
	defaultID, defaultPort, defaultAddr := nextInterfaceDefaults(existing)
	id, setID := Signal(defaultID)
	port, setPort := Signal(defaultPort)
	address, setAddress := Signal(defaultAddr)
	dns, setDNS := Signal("")
	endpoint, setEndpoint := Signal("")
	errMsg, setErrMsg := Signal("")

	doCreate := func() {
		setErrMsg("")
		if id() == "" {
			setErrMsg("Interface ID is required")
			return
		}
		if address() == "" {
			setErrMsg("Address is required")
			return
		}
		portNum, err := strconv.Atoi(port())
		if err != nil || portNum < 1 || portNum > 65535 {
			setErrMsg("Port must be a number between 1 and 65535")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/interfaces", map[string]any{
				"id":          id(),
				"type":        "client-access",
				"listen_port": portNum,
				"address":     address(),
				"dns":         dns(),
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
			onCreated()
		}()
	}

	return Card(
		CardHeader("New Interface"),

		ErrorAlert(errMsg),

		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Interface ID", "text", "wg0", id, func(v string) { setID(v) }),
			FormField("Listen Port", "number", "51820", port, func(v string) { setPort(v) }),
			FormField("Address (CIDR)", "text", "10.100.0.1/24", address, func(v string) { setAddress(v) }),
			FormField("DNS", "text", "1.1.1.1", dns, func(v string) { setDNS(v) }),
			FormField("Public Endpoint", "text", "vpn.example.com", endpoint, func(v string) { setEndpoint(v) }),
		),

		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Create", "primary", doCreate),
			Btn("Cancel", "ghost", func() {
				interfacesFormMode = "none"
				js.Global().Get("window").Get("history").Call("replaceState", nil, "", "/interfaces")
				refreshRoute()
			}),
		),
	)
}

func importInterfaceForm(onImported func()) loom.Node {
	id, setID := Signal("")
	config, setConfig := Signal("")
	errMsg, setErrMsg := Signal("")

	doImport := func() {
		setErrMsg("")
		if id() == "" {
			setErrMsg("Interface ID is required")
			return
		}
		if config() == "" {
			setErrMsg("Paste your WireGuard config")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/interfaces/import", map[string]any{
				"id":     id(),
				"type":   "client-access",
				"config": config(),
			}, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			onImported()
		}()
	}

	return Card(
		CardHeader("Import WireGuard Config"),

		ErrorAlert(errMsg),

		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Interface ID", "text", "wg0", id, func(v string) { setID(v) }),
			Div(), // empty cell for grid alignment
		),

		Div(
			Apply(Attr{"class": "mb-4"}),
			Elem("label", Apply(Attr{"class": "block text-xs font-medium text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("Configuration")),
			Elem("textarea",
				Apply(Attr{
					"class":       "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-lg text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/50 focus:ring-1 focus:ring-wg-600/20 font-mono transition-colors",
					"placeholder": "[Interface]\nPrivateKey = ...\nAddress = 10.200.0.1/30\nListenPort = 51820\n\n[Peer]\n# My Peer\nPublicKey = ...\nAllowedIPs = 10.200.0.2/32",
					"rows":        "12",
				}),
				Apply(On{"input": func(evt *EventInput) {
					setConfig(evt.InputValue())
				}}),
			),
		),

		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Import", "primary", doImport),
		),
	)
}
