//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

var peerConfigPollInterval js.Value

var privateKeyRe = regexp.MustCompile(`(?m)(PrivateKey\s*=\s*)(.+)$`)

func PeerConfigView(ifaceID, peerID string) loom.Node {
	conf, setConf := Signal("")
	configErr, setConfigErr := Signal("")
	loading, setLoading := Signal(true)
	peerName, setPeerName := Signal("")

	Effect(func() {
		go func() {
			type result struct {
				text string
				name string
				ok   bool
			}
			done := make(chan result, 1)

			opts := js.Global().Get("Object").New()
			opts.Set("method", "GET")
			opts.Set("credentials", "same-origin")

			thenFn := js.FuncOf(func(this js.Value, args []js.Value) any {
				response := args[0]
				isOK := response.Get("ok").Bool()
				name := response.Get("headers").Call("get", "X-Peer-Name").String()
				response.Call("text").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					done <- result{text: args[0].String(), name: name, ok: isOK}
					return nil
				}))
				return nil
			})
			defer thenFn.Release()

			catchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
				done <- result{text: "Failed to fetch config", ok: false}
				return nil
			})
			defer catchFn.Release()

			js.Global().Call("fetch", fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/config", ifaceID, peerID), opts).
				Call("then", thenFn).Call("catch", catchFn)

			r := <-done
			if r.ok {
				setConf(r.text)
				if r.name != "" && r.name != "<null>" {
					setPeerName(r.name)
				}
			} else {
				setConfigErr("This peer was imported without a private key. Client configuration is not available.")
			}
			setLoading(false)
		}()
	})

	// Poll for first connection — celebrate once per peer
	Effect(func() {
		lsKey := "wgrift:celebrated:" + peerID
		if js.Global().Get("localStorage").Call("getItem", lsKey).Truthy() {
			return // already celebrated
		}

		checkConnection := func() {
			go func() {
				var resp apiResponse
				if err := apiFetch("GET", fmt.Sprintf("/api/v1/interfaces/%s/status", ifaceID), nil, &resp); err != nil {
					return
				}
				var s interfaceStatusData
				if err := json.Unmarshal(resp.Data, &s); err != nil {
					return
				}
				for _, p := range s.Peers {
					if p.Peer.ID == peerID && p.Connected {
						js.Global().Get("localStorage").Call("setItem", lsKey, "1")
						name := p.Peer.Name
						if name == "" {
							name = peerName()
						}
						if name == "" {
							name = peerID[:8] + "..."
						}
						showCelebration(name)
						// Stop polling
						if !peerConfigPollInterval.IsUndefined() && !peerConfigPollInterval.IsNull() {
							js.Global().Call("clearInterval", peerConfigPollInterval)
							peerConfigPollInterval = js.Undefined()
						}
						return
					}
				}
			}()
		}

		checkConnection()
		if !peerConfigPollInterval.IsUndefined() && !peerConfigPollInterval.IsNull() {
			js.Global().Call("clearInterval", peerConfigPollInterval)
		}
		peerConfigPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
			checkConnection()
			return nil
		}), 3000)
	})

	downloadConf := func() {
		c := conf()
		if c == "" {
			return
		}
		// Create a blob and trigger download
		blob := js.Global().Get("Blob").New(
			js.Global().Get("Array").Call("of", c),
			map[string]any{"type": "text/plain"},
		)
		url := js.Global().Get("URL").Call("createObjectURL", blob)
		a := js.Global().Get("document").Call("createElement", "a")
		a.Set("href", url)
		a.Set("download", fmt.Sprintf("%s.conf", peerID))
		a.Call("click")
		js.Global().Get("URL").Call("revokeObjectURL", url)
	}

	showKey, setShowKey := Signal(false)
	copied, setCopied := Signal(false)
	configExpanded, setConfigExpanded := Signal(false)

	maskedConf := func() string {
		return privateKeyRe.ReplaceAllString(conf(), "${1}••••••••••••••••••••••••••••••••••••••••••••")
	}

	return Div(
		Div(
			Apply(Attr{"class": "flex flex-wrap items-center gap-1.5 mb-6"}),
			Button(
				Apply(Attr{"class": "flex items-center gap-1 text-ink-3 hover:text-wg-400 text-sm transition-colors"}),
				Apply(On{"click": func() { navigate(fmt.Sprintf("/interfaces/%s", ifaceID)) }}),
				Icon("chevron-left", 16),
				Text("Interfaces"),
			),
			Span(Apply(Attr{"class": "text-ink-4/40"}), Text("/")),
			Button(
				Apply(Attr{"class": "text-ink-3 hover:text-wg-400 text-sm font-mono transition-colors"}),
				Apply(On{"click": func() { navigate(fmt.Sprintf("/interfaces/%s", ifaceID)) }}),
				Text(ifaceID),
			),
			Span(Apply(Attr{"class": "text-ink-4/40"}), Text("/")),
			Bind(func() loom.Node {
				name := peerName()
				if name == "" {
					name = peerID[:8] + "..."
				}
				return Span(Apply(Attr{"class": "font-mono text-lg font-bold text-ink-1 tracking-tight"}), Text(name))
			}),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() && configErr() != "" }, func() loom.Node {
			return Card(
				Div(
					Apply(Attr{"class": "flex flex-col items-center gap-3 py-8 text-center"}),
					Div(Apply(Attr{"class": "text-ink-3 text-4xl"}), Text("🔒")),
					Div(Apply(Attr{"class": "text-ink-1 font-medium"}), Text("Config Unavailable")),
					Div(Apply(Attr{"class": "text-ink-3 text-sm max-w-md"}), Text(configErr())),
					Btn("← Back", "ghost", func() {
						navigate(fmt.Sprintf("/interfaces/%s", ifaceID))
					}),
				),
			)
		}),
		Show(func() bool { return !loading() && configErr() == "" }, func() loom.Node {
			return Div(
				Apply(Attr{"class": "grid grid-cols-1 lg:grid-cols-2 gap-6"}),

				// Config text
				Card(
					// Mobile: accordion header with action buttons
					Div(
						Apply(Attr{"class": "sm:hidden -m-6 p-6"}),
						Div(
							Apply(Attr{"class": "flex items-center justify-between"}),
							H3(
								Apply(Attr{"class": "text-[11px] font-semibold text-ink-3 uppercase tracking-[0.15em] cursor-pointer"}),
								Apply(On{"click": func() { setConfigExpanded(!configExpanded()) }}),
								Text("Configuration"),
							),
							Bind(func() loom.Node {
								chevronCls := "transition-transform duration-200 text-ink-3 cursor-pointer"
								if configExpanded() {
									chevronCls += " rotate-180"
								}
								return Span(
									Apply(Attr{"class": chevronCls}),
									Apply(On{"click": func() { setConfigExpanded(!configExpanded()) }}),
									Icon("chevron-down", 16),
								)
							}),
						),
						Div(
							Apply(Attr{"class": "grid grid-cols-3 gap-2 mt-3"}),
							Bind(func() loom.Node {
								isCopied := copied()
								btnClass := "inline-flex items-center justify-center gap-1.5 text-sm font-medium rounded-lg border cursor-pointer px-3 py-2.5 transition-all duration-300 "
								svg := icons["copy"](14)
								label := "Copy"
								if isCopied {
									btnClass += "border-green-500/30 text-green-400 bg-green-500/10 scale-105"
									svg = icons["check"](14)
									label = "Copied!"
								} else {
									btnClass += "border-line-1 text-ink-2 hover:bg-surface-3"
								}
								return Button(
									Apply(Attr{"class": btnClass}),
									Apply(On{"click": func() {
										if !copied() {
											js.Global().Get("navigator").Get("clipboard").Call("writeText", conf())
											setCopied(true)
											js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
												setCopied(false)
												return nil
											}), 2000)
										}
									}}),
									Span(Apply(innerHTML(svg))),
									Span(Text(label)),
								)
							}),
							Button(
								Apply(Attr{"class": "inline-flex items-center justify-center gap-1.5 text-sm font-medium rounded-lg border transition-all duration-150 cursor-pointer px-3 py-2.5 border-line-1 text-ink-2 hover:bg-surface-3"}),
								Apply(On{"click": func() { downloadConf() }}),
								Icon("download", 14),
								Span(Text("Download")),
							),
							Bind(func() loom.Node {
								cls := "inline-flex items-center justify-center gap-1.5 text-sm font-medium rounded-lg border transition-all duration-150 cursor-pointer px-3 py-2.5 border-line-1 text-ink-2 hover:bg-surface-3"
								if !smtpEnabled() {
									cls = "hidden"
								}
								return Button(
									Apply(Attr{"class": cls}),
									Apply(On{"click": func() {
										ShowEmailModal(peerName(), ifaceID, peerID)
									}}),
									Icon("mail", 14),
									Span(Text("Email")),
								)
							}),
						),
					),
					// Desktop: normal header with actions (hidden on mobile)
					Div(
						Apply(Attr{"class": "hidden sm:block"}),
						CardHeader("Configuration",
							Div(
								Apply(Attr{"class": "flex flex-wrap gap-2"}),
								Bind(func() loom.Node {
									isCopied := copied()
									btnClass := "inline-flex items-center gap-1.5 text-xs font-medium rounded-lg border cursor-pointer px-3 py-1.5 transition-all duration-300 "
									svg := icons["copy"](14)
									label := "Copy"
									if isCopied {
										btnClass += "border-green-500/30 text-green-400 bg-green-500/10 scale-105"
										svg = icons["check"](14)
										label = "Copied!"
									} else {
										btnClass += "border-line-1 text-ink-2 hover:bg-surface-3"
									}
									return Button(
										Apply(Attr{"class": btnClass}),
										Apply(On{"click": func() {
											if !copied() {
												js.Global().Get("navigator").Get("clipboard").Call("writeText", conf())
												setCopied(true)
												js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
													setCopied(false)
													return nil
												}), 2000)
											}
										}}),
										Span(Apply(innerHTML(svg))),
										Span(Text(label)),
									)
								}),
								Button(
									Apply(Attr{"class": "inline-flex items-center gap-1.5 text-xs font-medium rounded-lg border transition-all duration-150 cursor-pointer px-3 py-1.5 border-line-1 text-ink-2 hover:bg-surface-3"}),
									Apply(On{"click": func() { downloadConf() }}),
									Icon("download", 14),
									Span(Text("Download")),
								),
								Bind(func() loom.Node {
									cls := "inline-flex items-center gap-1.5 text-xs font-medium rounded-lg border transition-all duration-150 cursor-pointer px-3 py-1.5 border-line-1 text-ink-2 hover:bg-surface-3"
									if !smtpEnabled() {
										cls = "hidden"
									}
									return Button(
										Apply(Attr{"class": cls}),
										Apply(On{"click": func() {
											ShowEmailModal(peerName(), ifaceID, peerID)
										}}),
										Icon("mail", 14),
										Span(Text("Email")),
									)
								}),
							),
						),
					),
					// Config body: hidden on mobile until expanded, always visible on desktop
					Bind(func() loom.Node {
						bodyCls := "hidden sm:block"
						if configExpanded() {
							bodyCls = "block mt-4 sm:mt-0"
						}
						display := maskedConf()
						if showKey() {
							display = conf()
						}
						return Div(
							Apply(Attr{"class": bodyCls}),
							Elem("pre",
								Apply(Attr{"class": "font-mono text-xs text-ink-2 bg-surface-2 border border-line-1 rounded-lg p-5 overflow-auto whitespace-pre leading-relaxed"}),
								Text(display),
							),
							Div(
								Apply(Attr{"class": "mt-3 flex items-center gap-2"}),
								Bind(func() loom.Node {
									svg := icons["eye"](14)
									label := "Reveal private key"
									if showKey() {
										svg = icons["eye-off"](14)
										label = "Hide private key"
									}
									return Button(
										Apply(Attr{"class": "inline-flex items-center gap-1.5 text-xs text-ink-3 hover:text-ink-1 transition-colors"}),
										Apply(On{"click": func() { setShowKey(!showKey()) }}),
										Span(Apply(innerHTML(svg))),
										Span(Text(label)),
									)
								}),
							),
						)
					}),
				),

				// QR Code
				Card(
					CardHeader("QR Code"),
					Div(
						Apply(Attr{"class": "flex justify-center"}),
						Img(Apply(Attr{
							"src":   fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/qr", ifaceID, peerID),
							"alt":   "WireGuard QR Code",
							"class": "max-w-[256px] rounded-lg bg-white p-2",
						})),
					),
				),

				// Setup instructions + app links
				Div(
					Apply(Attr{"class": "lg:col-span-2"}),
					Card(
						CardHeader("Setup Instructions"),
						Div(
							Apply(Attr{"class": "space-y-4 text-sm text-ink-2"}),
							setupStep("1", "Install WireGuard", "Download and install the WireGuard app for your platform."),
							Div(
								Apply(Attr{"class": "grid grid-cols-2 sm:grid-cols-4 gap-3 ml-9 mr-9"}),
								appLink("macOS", "https://itunes.apple.com/us/app/wireguard/id1451685025?ls=1&mt=12"),
								appLink("iOS", "https://itunes.apple.com/us/app/wireguard/id1441195209?ls=1&mt=8"),
								appLink("Android", "https://play.google.com/store/apps/details?id=com.wireguard.android"),
								appLink("Windows", "https://download.wireguard.com/windows-client/wireguard-installer.exe"),
							),
							setupStep("2", "Add Tunnel", "Open WireGuard, tap + or \"Add Tunnel\", then either scan the QR code above or import the downloaded .conf file."),
							setupStep("3", "Activate", "Toggle the tunnel on. You should see a handshake within a few seconds confirming the connection is active."),
						),
					),
				),
			)
		}),
	)
}

func setupStep(number, title, description string) loom.Node {
	return Div(
		Apply(Attr{"class": "flex gap-3"}),
		Div(
			Apply(Attr{"class": "flex-shrink-0 w-6 h-6 rounded-full bg-wg-600/15 text-wg-400 text-xs font-bold flex items-center justify-center mt-0.5"}),
			Text(number),
		),
		Div(
			Div(Apply(Attr{"class": "font-medium text-ink-1"}), Text(title)),
			Div(Apply(Attr{"class": "text-ink-3 mt-0.5"}), Text(description)),
		),
	)
}

func appLink(label, url string) loom.Node {
	return Elem("a",
		Apply(Attr{
			"href":   url,
			"target": "_blank",
			"rel":    "noopener noreferrer",
			"class":  "flex items-center justify-center gap-2 px-4 py-3 bg-surface-2 border border-line-1 rounded-lg text-sm font-medium text-ink-2 hover:text-wg-500 hover:border-wg-500/40 transition-colors",
		}),
		Text(label),
		Icon("external-link", 14),
	)
}
