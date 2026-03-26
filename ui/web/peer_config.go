//go:build js && wasm

package main

import (
	"fmt"
	"regexp"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

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

	maskedConf := func() string {
		return privateKeyRe.ReplaceAllString(conf(), "${1}••••••••••••••••••••••••••••••••••••••••••••")
	}

	return Div(
		Div(
			Apply(Attr{"class": "flex items-center gap-3 mb-6"}),
			Button(
				Apply(Attr{"class": "flex items-center gap-1 text-gray-400 hover:text-gray-700 text-sm transition-colors"}),
				Apply(On{"click": func() { navigate(fmt.Sprintf("/interfaces/%s", ifaceID)) }}),
				Icon("chevron-left", 16),
				Text("Interfaces"),
			),
			Span(Apply(Attr{"class": "text-gray-300"}), Text("/")),
			Button(
				Apply(Attr{"class": "text-gray-400 hover:text-gray-700 text-sm font-mono transition-colors"}),
				Apply(On{"click": func() { navigate(fmt.Sprintf("/interfaces/%s", ifaceID)) }}),
				Text(ifaceID),
			),
			Span(Apply(Attr{"class": "text-gray-300"}), Text("/")),
			Bind(func() loom.Node {
				name := peerName()
				if name == "" {
					name = peerID[:8] + "..."
				}
				return Span(Apply(Attr{"class": "font-mono text-lg font-semibold text-gray-900"}), Text(name))
			}),
		),

		LoadingView(loading),
		Show(func() bool { return !loading() && configErr() != "" }, func() loom.Node {
			return Card(
				Div(
					Apply(Attr{"class": "flex flex-col items-center gap-3 py-8 text-center"}),
					Div(Apply(Attr{"class": "text-gray-400 text-4xl"}), Text("🔒")),
					Div(Apply(Attr{"class": "text-gray-600 font-medium"}), Text("Config Unavailable")),
					Div(Apply(Attr{"class": "text-gray-400 text-sm max-w-md"}), Text(configErr())),
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
					CardHeader("Configuration",
						Div(
							Apply(Attr{"class": "flex gap-2"}),
							Bind(func() loom.Node {
							isCopied := copied()
							btnClass := "inline-flex items-center gap-1.5 text-xs font-medium rounded-md border cursor-pointer px-3 py-1.5 transition-all duration-300 "
							svg := icons["copy"](14)
							label := "Copy"
							if isCopied {
								btnClass += "border-emerald-300 text-emerald-600 bg-emerald-50 scale-105"
								svg = icons["check"](14)
								label = "Copied!"
							} else {
								btnClass += "border-gray-300 text-gray-600 hover:bg-gray-50"
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
							Apply(Attr{"class": "inline-flex items-center gap-1.5 text-xs font-medium rounded-md border transition-colors cursor-pointer px-3 py-1.5 border-gray-300 text-gray-600 hover:bg-gray-50"}),
							Apply(On{"click": func() { downloadConf() }}),
							Icon("download", 14),
							Span(Text("Download")),
						),
						),
					),
					Bind(func() loom.Node {
						display := maskedConf()
						if showKey() {
							display = conf()
						}
						return Elem("pre",
							Apply(Attr{"class": "font-mono text-xs text-gray-600 bg-gray-100 rounded p-4 overflow-auto whitespace-pre"}),
							Text(display),
						)
					}),
					Div(
						Apply(Attr{"class": "mt-3 flex items-center gap-2"}),
						Bind(func() loom.Node {
							if showKey() {
								return Button(
									Apply(Attr{"class": "inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 transition-colors"}),
									Apply(On{"click": func() { setShowKey(false) }}),
									Icon("eye-off", 14),
									Text("Hide private key"),
								)
							}
							return Button(
								Apply(Attr{"class": "inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-gray-700 transition-colors"}),
								Apply(On{"click": func() { setShowKey(true) }}),
								Icon("eye", 14),
								Text("Reveal private key"),
							)
						}),
					),
				),

				// QR Code
				Card(
					CardHeader("QR Code"),
					Div(
						Apply(Attr{"class": "flex justify-center"}),
						Img(Apply(Attr{
							"src":   fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/qr", ifaceID, peerID),
							"alt":   "WireGuard QR Code",
							"class": "max-w-[256px] rounded bg-white p-2",
						})),
					),
				),
			)
		}),
	)
}
