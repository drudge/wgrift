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

func InterfacesView() loom.Node {
	ifaces, setIfaces := Signal[[]interfaceData](nil)
	loading, setLoading := Signal(true)
	// "none", "create", "import"
	initialForm := "none"
	search := js.Global().Get("window").Get("location").Get("search").String()
	if strings.Contains(search, "action=create") {
		initialForm = "create"
	} else if strings.Contains(search, "action=import") {
		initialForm = "import"
	}
	showForm, setShowForm := Signal(initialForm)

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
		Div(
			Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-8"}),
			H2(Apply(Attr{"class": "text-xl font-semibold text-gray-900"}), Text("Interfaces")),
			Div(
				Apply(Attr{"class": "flex gap-2"}),
				Btn("Import Config", "ghost", func() {
					if showForm() == "import" {
						setShowForm("none")
					} else {
						setShowForm("import")
					}
				}),
				Btn("Create Interface", "primary", func() {
					if showForm() == "create" {
						setShowForm("none")
					} else {
						setShowForm("create")
					}
				}),
			),
		),

		// Form area — wrapped in Bind with same-structure branches
		Bind(func() loom.Node {
			form := showForm()
			if form == "create" {
				return Div(
					Apply(Attr{"class": "mb-4"}),
					createInterfaceForm(func() { refreshRoute() }),
				)
			}
			if form == "import" {
				return Div(
					Apply(Attr{"class": "mb-4"}),
					importInterfaceForm(func() { refreshRoute() }),
				)
			}
			return Div(
				Apply(Attr{"class": "hidden"}),
				Div(),
			)
		}),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			list := ifaces()
			if len(list) == 0 {
				return EmptyState("No interfaces configured")
			}

			// Mobile cards
			cards := make([]loom.Node, 0, len(list))
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

				cards = append(cards, Div(
					Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-4 active:bg-gray-50"}),
					Apply(On{"click": clickNav}),
					Div(
						Apply(Attr{"class": "flex items-center justify-between mb-2"}),
						Span(Apply(Attr{"class": "font-mono text-sm font-medium text-gray-900"}), Text(iface.ID)),
						badge,
					),
					Div(
						Apply(Attr{"class": "font-mono text-xs text-gray-400"}),
						Text(fmt.Sprintf("%s · port %d", iface.Address, iface.ListenPort)),
					),
				))

				rows = append(rows, Elem("tr",
					Apply(Attr{"class": "border-b border-gray-100 hover:bg-gray-50 cursor-pointer"}),
					Apply(On{"click": clickNav}),
					Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-sm font-medium text-gray-900"}), Text(iface.ID)),
					Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-sm text-gray-500"}), Text(iface.Address)),
					Elem("td", Apply(Attr{"class": "px-4 py-3 font-mono text-sm text-gray-500"}), Text(fmt.Sprintf("%d", iface.ListenPort))),
					Elem("td", Apply(Attr{"class": "px-4 py-3"}), func() loom.Node {
						if iface.Enabled {
							return Badge("enabled", "emerald")
						}
						return Badge("disabled", "")
					}()),
				))
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
								Apply(Attr{"class": "border-b border-gray-200 text-left text-xs uppercase tracking-wider text-gray-400"}),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("ID")),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Address")),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Port")),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Status")),
							),
						),
						Elem("tbody", rows...),
					),
				),
			)
		}),
	)
}

func createInterfaceForm(onCreated func()) loom.Node {
	id, setID := Signal("")
	port, setPort := Signal("")
	address, setAddress := Signal("")
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
			Elem("label", Apply(Attr{"class": "block text-sm text-gray-500 mb-1"}), Text("Configuration")),
			Elem("textarea",
				Apply(Attr{
					"class":       "w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-gray-900 text-sm placeholder-gray-400 focus:outline-none focus:border-teal-500 focus:ring-1 focus:ring-teal-500/20 font-mono",
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
