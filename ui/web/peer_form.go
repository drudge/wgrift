//go:build js && wasm

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

type peerLabels struct {
	headerAdd, headerEdit string
	namePlaceholder       string
	serverIPsLabel        string
	serverIPsPlaceholder  string
	serverIPsHelp         string
	clientIPsLabel        string
	clientIPsPlaceholder  string
	clientIPsHelp         string
	keepaliveHelp         string
	submitAdd             string
}

func labelsForType(peerType string) peerLabels {
	if peerType == "site" {
		return peerLabels{
			headerAdd:            "Add Site",
			headerEdit:           "Edit Site",
			namePlaceholder:      "Branch Office, Data Center, etc.",
			serverIPsLabel:       "Remote Subnets",
			serverIPsPlaceholder: "Networks behind this site (e.g. 192.168.1.0/24)",
			serverIPsHelp:        "Subnets reachable through this site's gateway",
			clientIPsLabel:       "Routes to Advertise",
			clientIPsPlaceholder: "Subnets this side shares with the remote site",
			clientIPsHelp:        "Networks on your side the remote site should reach",
			keepaliveHelp:        "Required if either side is behind NAT",
			submitAdd:            "Add Site",
		}
	}
	return peerLabels{
		headerAdd:            "Add Client",
		headerEdit:           "Edit Client",
		namePlaceholder:      "Phone, Laptop, etc.",
		serverIPsLabel:       "Server Allowed IPs",
		serverIPsPlaceholder: "Subnets behind this peer",
		serverIPsHelp:        "Auto-filled with tunnel IP if left empty",
		clientIPsLabel:       "Client Allowed IPs",
		clientIPsPlaceholder: "Empty = route all traffic through tunnel",
		clientIPsHelp:        "Leave empty for full tunnel. Add subnets for split tunnel.",
		keepaliveHelp:        "Keeps connection alive through NAT",
		submitAdd:            "Add Client",
	}
}

func PeerEditForm(ifaceID string, peer peerData, onSaved func()) loom.Node {
	initialType := peer.Type
	if initialType == "" {
		initialType = "client"
	}
	editPeerType, setEditPeerType := Signal(initialType)
	name, setName := Signal(peer.Name)
	address, setAddress := Signal(peer.Address)
	dns, setDNS := Signal(peer.DNS)
	kaDefault := "25"
	if peer.PersistentKeepalive > 0 {
		kaDefault = strconv.Itoa(peer.PersistentKeepalive)
	}
	keepalive, setKeepalive := Signal(kaDefault)
	errMsg, setErrMsg := Signal("")
	FocusInput(`input[placeholder="Phone, Laptop, etc."]`)

	// Parse existing server-side allowed IPs into chips
	var allowedIPs []string
	for _, ip := range strings.Split(peer.AllowedIPs, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			allowedIPs = append(allowedIPs, ip)
		}
	}

	// Parse existing client-side allowed IPs into chips
	var clientAllowedIPs []string
	for _, ip := range strings.Split(peer.ClientAllowedIPs, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			clientAllowedIPs = append(clientAllowedIPs, ip)
		}
	}

	chipsID := "edit-allowed-ips-chips"
	inputID := "edit-allowed-ips-input"
	clientChipsID := "edit-client-allowed-ips-chips"
	clientInputID := "edit-client-allowed-ips-input"

	var doRenderChips func()
	doRenderChips = func() {
		renderChipList(chipsID, allowedIPs, func(idx int) {
			allowedIPs = append(allowedIPs[:idx], allowedIPs[idx+1:]...)
			doRenderChips()
		})
	}

	var doRenderClientChips func()
	doRenderClientChips = func() {
		renderChipList(clientChipsID, clientAllowedIPs, func(idx int) {
			clientAllowedIPs = append(clientAllowedIPs[:idx], clientAllowedIPs[idx+1:]...)
			doRenderClientChips()
		})
	}

	// Render chips after mount
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		doRenderChips()
		doRenderClientChips()
		return nil
	}), 0)

	addIP := func(raw string) {
		ip := strings.TrimSpace(raw)
		ip = strings.TrimRight(ip, ",")
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		if _, _, err := net.ParseCIDR(ip); err != nil {
			setErrMsg(fmt.Sprintf("Invalid CIDR: %s", ip))
			return
		}
		for _, existing := range allowedIPs {
			if existing == ip {
				return
			}
		}
		allowedIPs = append(allowedIPs, ip)
		doRenderChips()
		el := js.Global().Get("document").Call("getElementById", inputID)
		if el.Truthy() {
			el.Set("value", "")
		}
	}

	addClientIP := func(raw string) {
		ip := strings.TrimSpace(raw)
		ip = strings.TrimRight(ip, ",")
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		if _, _, err := net.ParseCIDR(ip); err != nil {
			setErrMsg(fmt.Sprintf("Invalid CIDR: %s", ip))
			return
		}
		for _, existing := range clientAllowedIPs {
			if existing == ip {
				return
			}
		}
		clientAllowedIPs = append(clientAllowedIPs, ip)
		doRenderClientChips()
		el := js.Global().Get("document").Call("getElementById", clientInputID)
		if el.Truthy() {
			el.Set("value", "")
		}
	}

	doSave := func() {
		setErrMsg("")
		// Flush pending input for server IPs
		el := js.Global().Get("document").Call("getElementById", inputID)
		if el.Truthy() {
			val := el.Get("value").String()
			if strings.TrimSpace(val) != "" {
				addIP(val)
			}
		}
		// Flush pending input for client IPs
		el2 := js.Global().Get("document").Call("getElementById", clientInputID)
		if el2.Truthy() {
			val := el2.Get("value").String()
			if strings.TrimSpace(val) != "" {
				addClientIP(val)
			}
		}
		// Auto-add tunnel IP as /32 to server AllowedIPs if empty
		if len(allowedIPs) == 0 && address() != "" {
			tunnelIP := address()
			if ip, _, err := net.ParseCIDR(tunnelIP); err == nil {
				tunnelIP = ip.String() + "/32"
			} else if ip := net.ParseIP(tunnelIP); ip != nil {
				tunnelIP = ip.String() + "/32"
			}
			allowedIPs = append(allowedIPs, tunnelIP)
			doRenderChips()
		}
		if len(allowedIPs) == 0 {
			setErrMsg("Tunnel address is required")
			return
		}
		go func() {
			ips := strings.Join(allowedIPs, ", ")
			clientIPs := strings.Join(clientAllowedIPs, ", ")
			body := map[string]any{
				"type":                 editPeerType(),
				"name":                 name(),
				"address":              address(),
				"allowed_ips":          ips,
				"client_allowed_ips":   clientIPs,
				"dns":                  dns(),
				"persistent_keepalive": 0,
			}
			if keepalive() != "" {
				if ka, err := strconv.Atoi(keepalive()); err == nil {
					body["persistent_keepalive"] = ka
				}
			}

			var resp apiResponse
			err := apiFetch("PUT", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s", ifaceID, peer.ID), body, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			onSaved()
		}()
	}

	return Card(
		Bind(func() loom.Node {
			lbl := labelsForType(editPeerType())
			return Span(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text(lbl.headerEdit))
		}),

		ErrorAlert(errMsg),

		// Type selector
		Bind(func() loom.Node {
			t := editPeerType()
			return Div(
				Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4 mb-5"}),
				typeSelectorCard("laptop", "Client", "Remote device connecting to your network", t == "client", func() { setEditPeerType("client") }),
				typeSelectorCard("building-2", "Site", "Gateway connecting two networks together", t == "site", func() { setEditPeerType("site") }),
			)
		}),

		Bind(func() loom.Node {
			lbl := labelsForType(editPeerType())
			return Div(
				Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
				FormField("Name", "text", lbl.namePlaceholder, name, func(v string) { setName(v) }),
				FormField("Tunnel Address", "text", "10.100.0.2/32", address, func(v string) { setAddress(v) }),
				chipInput(lbl.serverIPsLabel, lbl.serverIPsPlaceholder, lbl.serverIPsHelp, chipsID, inputID, addIP, func() []string { return allowedIPs }, func() {
					if len(allowedIPs) > 0 {
						allowedIPs = allowedIPs[:len(allowedIPs)-1]
						doRenderChips()
					}
				}),
				chipInput(lbl.clientIPsLabel, lbl.clientIPsPlaceholder, lbl.clientIPsHelp, clientChipsID, clientInputID, addClientIP, func() []string { return clientAllowedIPs }, func() {
					if len(clientAllowedIPs) > 0 {
						clientAllowedIPs = clientAllowedIPs[:len(clientAllowedIPs)-1]
						doRenderClientChips()
					}
				}),
				FormField("DNS", "text", "Override interface DNS (optional)", dns, func(v string) { setDNS(v) }),
				FormFieldWithHelp("Keepalive (sec)", "number", "25", lbl.keepaliveHelp, keepalive, func(v string) { setKeepalive(v) }),
			)
		}),

		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Save", "primary", doSave),
			Btn("Cancel", "ghost", onSaved),
		),
	)
}

// renderChipList renders CIDR chip elements into a DOM container by ID.
func renderChipList(containerID string, items []string, onRemove func(int)) {
	container := js.Global().Get("document").Call("getElementById", containerID)
	if !container.Truthy() {
		return
	}
	container.Set("innerHTML", "")
	for i, ip := range items {
		idx := i
		chip := js.Global().Get("document").Call("createElement", "span")
		chip.Set("className", "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-mono bg-surface-3 border border-line-1 text-ink-2")
		chip.Set("innerHTML", ip+` <button class="text-ink-4 hover:text-red-400 ml-0.5 transition-colors">&times;</button>`)
		chip.Call("querySelector", "button").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			onRemove(idx)
			return nil
		}))
		container.Call("appendChild", chip)
	}
}

// chipInput renders a label + chip container + text input for entering CIDRs.
func chipInput(label, placeholder, helpText, chipsID, inputID string, addFn func(string), _ func() []string, onBackspace func()) loom.Node {
	helpClass := "text-xs text-ink-4 mt-1.5"
	if helpText == "" {
		helpClass = "hidden"
	}
	return Div(
		Apply(Attr{"class": "mb-4"}),
		Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text(label)),
		Div(
			Apply(Attr{"class": "w-full px-3 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm focus-within:border-wg-600/40 focus-within:ring-1 focus-within:ring-wg-600/15 transition-colors"}),
			Div(Apply(Attr{"id": chipsID, "class": "flex flex-wrap gap-1"})),
			Input(
				Apply(Attr{
					"id":          inputID,
					"class":       "w-full bg-transparent text-ink-1 text-sm placeholder-ink-4 focus:outline-none font-mono mt-1",
					"placeholder": placeholder + " — Enter to add",
				}),
				Apply(On{"keydown": func(evt *EventKeyboard) {
					key := evt.Key()
					val := evt.Target().Get("value").String()
					if key == "Enter" || key == "," || key == "Tab" {
						evt.PreventDefault()
						addFn(val)
					}
					if key == "Backspace" && val == "" {
						onBackspace()
					}
				}}),
				Apply(On{"paste": func(evt *Event) {
					js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
						el := js.Global().Get("document").Call("getElementById", inputID)
						if !el.Truthy() {
							return nil
						}
						val := el.Get("value").String()
						for _, p := range strings.Split(val, ",") {
							addFn(p)
						}
						return nil
					}), 0)
				}}),
			),
		),
		P(Apply(Attr{"class": helpClass}), Text(helpText)),
	)
}

// nextAvailableIP computes the next unused IP in the interface's subnet.
func nextAvailableIP(ifaceAddr string, usedAddrs []string) string {
	// Parse interface CIDR (e.g. "10.200.0.1/24")
	ip, ipNet, err := net.ParseCIDR(ifaceAddr)
	if err != nil {
		return ""
	}
	gatewayIP := ip.To4()
	if gatewayIP == nil {
		return "" // IPv6 not supported yet
	}

	// Build set of used IPs (strip CIDR suffix)
	used := make(map[string]bool)
	used[gatewayIP.String()] = true // gateway itself is used
	for _, addr := range usedAddrs {
		a, _, err := net.ParseCIDR(addr)
		if err != nil {
			a = net.ParseIP(addr)
		}
		if a != nil {
			used[a.To4().String()] = true
		}
	}

	// Iterate through subnet, skip network and broadcast
	ones, bits := ipNet.Mask.Size()
	hostCount := 1 << (bits - ones)

	// Start from network+1
	base := make(net.IP, 4)
	copy(base, ipNet.IP.To4())

	for i := 1; i < hostCount-1; i++ {
		candidate := make(net.IP, 4)
		copy(candidate, base)
		candidate[0] += byte(i >> 24)
		candidate[1] += byte(i >> 16)
		candidate[2] += byte(i >> 8)
		candidate[3] += byte(i)
		if !used[candidate.String()] {
			return candidate.String() + "/32"
		}
	}
	return ""
}

func PeerForm(ifaceID string, ifaceAddr string, usedAddrs []string, onCreated func()) loom.Node {
	peerType, setPeerType := Signal("client")
	name, setName := Signal("")
	address, setAddress := Signal(nextAvailableIP(ifaceAddr, usedAddrs))
	dns, setDNS := Signal("")
	keepalive, setKeepalive := Signal("25")
	psk, setPSK := Signal(false)
	errMsg, setErrMsg := Signal("")
	FocusInput(`input[placeholder="Phone, Laptop, etc."]`)

	var allowedIPs []string
	var clientAllowedIPs []string
	chipsID := "allowed-ips-chips"
	inputID := "allowed-ips-input"
	clientChipsID := "client-allowed-ips-chips"
	clientInputID := "client-allowed-ips-input"

	var doRenderChips func()
	doRenderChips = func() {
		renderChipList(chipsID, allowedIPs, func(idx int) {
			allowedIPs = append(allowedIPs[:idx], allowedIPs[idx+1:]...)
			doRenderChips()
		})
	}

	var doRenderClientChips func()
	doRenderClientChips = func() {
		renderChipList(clientChipsID, clientAllowedIPs, func(idx int) {
			clientAllowedIPs = append(clientAllowedIPs[:idx], clientAllowedIPs[idx+1:]...)
			doRenderClientChips()
		})
	}

	addIP := func(raw string) {
		ip := strings.TrimSpace(raw)
		ip = strings.TrimRight(ip, ",")
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		if _, _, err := net.ParseCIDR(ip); err != nil {
			setErrMsg(fmt.Sprintf("Invalid CIDR: %s", ip))
			return
		}
		for _, existing := range allowedIPs {
			if existing == ip {
				return
			}
		}
		allowedIPs = append(allowedIPs, ip)
		doRenderChips()
		el := js.Global().Get("document").Call("getElementById", inputID)
		if el.Truthy() {
			el.Set("value", "")
		}
	}

	addClientIP := func(raw string) {
		ip := strings.TrimSpace(raw)
		ip = strings.TrimRight(ip, ",")
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		if _, _, err := net.ParseCIDR(ip); err != nil {
			setErrMsg(fmt.Sprintf("Invalid CIDR: %s", ip))
			return
		}
		for _, existing := range clientAllowedIPs {
			if existing == ip {
				return
			}
		}
		clientAllowedIPs = append(clientAllowedIPs, ip)
		doRenderClientChips()
		el := js.Global().Get("document").Call("getElementById", clientInputID)
		if el.Truthy() {
			el.Set("value", "")
		}
	}

	doAdd := func() {
		setErrMsg("")
		el := js.Global().Get("document").Call("getElementById", inputID)
		if el.Truthy() {
			val := el.Get("value").String()
			if strings.TrimSpace(val) != "" {
				addIP(val)
			}
		}
		el2 := js.Global().Get("document").Call("getElementById", clientInputID)
		if el2.Truthy() {
			val := el2.Get("value").String()
			if strings.TrimSpace(val) != "" {
				addClientIP(val)
			}
		}
		// Auto-add tunnel IP as /32 to server AllowedIPs if empty
		if len(allowedIPs) == 0 && address() != "" {
			tunnelIP := address()
			// Ensure it's a /32
			if ip, _, err := net.ParseCIDR(tunnelIP); err == nil {
				tunnelIP = ip.String() + "/32"
			} else if ip := net.ParseIP(tunnelIP); ip != nil {
				tunnelIP = ip.String() + "/32"
			}
			allowedIPs = append(allowedIPs, tunnelIP)
			doRenderChips()
		}
		if len(allowedIPs) == 0 {
			setErrMsg("Tunnel address is required")
			return
		}
		go func() {
			ips := strings.Join(allowedIPs, ", ")
			clientIPs := strings.Join(clientAllowedIPs, ", ")
			body := map[string]any{
				"type":               peerType(),
				"name":               name(),
				"address":            address(),
				"allowed_ips":        ips,
				"client_allowed_ips": clientIPs,
				"dns":                dns(),
				"psk":                psk(),
			}
			if keepalive() != "" {
				if ka, err := strconv.Atoi(keepalive()); err == nil {
					body["persistent_keepalive"] = ka
				}
			}

			var resp apiResponse
			err := apiFetch("POST", fmt.Sprintf("/api/v1/interfaces/%s/peers", ifaceID), body, &resp)
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
		Bind(func() loom.Node {
			lbl := labelsForType(peerType())
			return Span(Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text(lbl.headerAdd))
		}),

		ErrorAlert(errMsg),

		// Type selector
		Bind(func() loom.Node {
			t := peerType()
			return Div(
				Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4 mb-5"}),
				typeSelectorCard("laptop", "Client", "Remote device connecting to your network", t == "client", func() { setPeerType("client") }),
				typeSelectorCard("building-2", "Site", "Gateway connecting two networks together", t == "site", func() { setPeerType("site") }),
			)
		}),

		Bind(func() loom.Node {
			lbl := labelsForType(peerType())
			return Div(
				Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
				FormField("Name", "text", lbl.namePlaceholder, name, func(v string) { setName(v) }),
				FormField("Tunnel Address", "text", "10.100.0.2/32", address, func(v string) { setAddress(v) }),
				chipInput(lbl.serverIPsLabel, lbl.serverIPsPlaceholder, lbl.serverIPsHelp, chipsID, inputID, addIP, func() []string { return allowedIPs }, func() {
					if len(allowedIPs) > 0 {
						allowedIPs = allowedIPs[:len(allowedIPs)-1]
						doRenderChips()
					}
				}),
				chipInput(lbl.clientIPsLabel, lbl.clientIPsPlaceholder, lbl.clientIPsHelp, clientChipsID, clientInputID, addClientIP, func() []string { return clientAllowedIPs }, func() {
					if len(clientAllowedIPs) > 0 {
						clientAllowedIPs = clientAllowedIPs[:len(clientAllowedIPs)-1]
						doRenderClientChips()
					}
				}),
				FormField("DNS", "text", "Override interface DNS (optional)", dns, func(v string) { setDNS(v) }),
				FormFieldWithHelp("Keepalive (sec)", "number", "25", lbl.keepaliveHelp, keepalive, func(v string) { setKeepalive(v) }),
			)
		}),

		Div(
			Apply(Attr{"class": "flex flex-wrap items-center gap-4 mt-2"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2"}),
				Input(
					Apply(Attr{"type": "checkbox", "id": "psk-check", "class": "accent-wg-600"}),
					Apply(On{"change": func(evt *Event) {
						checked := evt.Target().Get("checked").Bool()
						setPSK(checked)
					}}),
				),
				Elem("label",
					Apply(Attr{"for": "psk-check", "class": "text-sm text-ink-2"}),
					Text("Generate preshared key"),
				),
			),
			Div(Apply(Attr{"class": "flex-1"})),
			Bind(func() loom.Node {
				lbl := labelsForType(peerType())
				return Btn(lbl.submitAdd, "primary", doAdd)
			}),
		),
	)
}
