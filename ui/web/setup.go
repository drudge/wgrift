//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

func SetupView() loom.Node {
	username, setUsername := Signal("")
	password, setPassword := Signal("")
	confirm, setConfirm := Signal("")
	errMsg, setErrMsg := Signal("")
	loading, setLoading := Signal(false)

	doSetup := func() {
		if loading() {
			return
		}
		if password() != confirm() {
			setErrMsg("Passwords do not match")
			return
		}
		if len(password()) < 16 {
			setErrMsg("Password must be at least 16 characters")
			return
		}
		setErrMsg("")
		setLoading(true)

		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/setup", map[string]string{
				"username": username(),
				"password": password(),
			}, &resp)

			if err != nil {
				setErrMsg("Setup failed. Check password requirements.")
				setLoading(false)
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				setLoading(false)
				return
			}

			var session sessionData
			if err := unmarshalData(resp.Data, &session); err == nil {
				js.Global().Get("window").Get("location").Call("reload")
				return
			}
			setLoading(false)
		}()
	}

	return Div(
		Apply(Attr{"class": "flex min-h-screen bg-surface-0"}),

		// Left brand panel
		Div(
			Apply(Attr{"class": "hidden md:flex w-[45%] bg-surface-1 border-r border-line-1 relative overflow-hidden flex-col items-center justify-center"}),
			Div(Apply(Attr{"class": "absolute w-64 h-64 bg-wg-600/8 rounded-full blur-3xl"})),
			Div(
				Apply(Attr{"class": "relative z-10 text-center"}),
				Div(
					Apply(Attr{"class": "text-5xl font-extrabold tracking-tight"}),
					Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
					Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
				),
				P(Apply(Attr{"class": "text-ink-3 text-sm mt-4 tracking-wide"}), Text("WireGuard VPN Management")),
			),
			Div(Apply(Attr{"class": "absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-wg-600/40 to-transparent"})),
		),

		// Right form panel
		Div(
			Apply(Attr{"class": "flex-1 flex items-center justify-center px-6"}),
			Div(
				Apply(Attr{"class": "w-full max-w-sm"}),

				// Mobile-only brand
				Div(
					Apply(Attr{"class": "md:hidden text-center mb-10"}),
					Div(
						Apply(Attr{"class": "text-4xl font-extrabold tracking-tight"}),
						Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
						Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
					),
					P(Apply(Attr{"class": "text-ink-3 text-sm mt-3"}), Text("WireGuard VPN Management")),
				),

				// Setup heading
				Div(
					Apply(Attr{"class": "mb-8"}),
					H2(Apply(Attr{"class": "text-2xl font-bold text-ink-1 tracking-tight"}), Text("Welcome")),
					P(Apply(Attr{"class": "text-ink-3 text-sm mt-2"}), Text("Create your admin account to get started")),
				),

				// Form
				Div(
					Apply(Attr{"class": "space-y-1"}),
					ErrorAlert(errMsg),
					FormField("Username", "text", "admin", username, func(v string) { setUsername(v) }),
					FormField("Password", "password", "min 16 characters", password, func(v string) { setPassword(v) }),
					FormField("Confirm Password", "password", "", confirm, func(v string) { setConfirm(v) }),
				),

				Button(
					Apply(Attr{"class": "w-full mt-6 px-4 py-3 text-sm font-semibold rounded-lg border border-wg-600/50 text-wg-400 bg-wg-600/5 hover:bg-wg-600/15 hover:border-wg-600/70 active:bg-wg-600/20 transition-all duration-100"}),
					Apply(On{"click": func() { doSetup() }}),
					Bind(func() loom.Node {
						if loading() {
							return Text("Creating account...")
						}
						return Text("Create Admin Account")
					}),
				),
			),
		),
	)
}
