//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

func LoginView() loom.Node {
	username, setUsername := Signal("")
	password, setPassword := Signal("")
	errMsg, setErrMsg := Signal(ErrorInfo{})
	loading, setLoading := Signal(false)

	hasOIDC := len(preloadOIDCProviders) > 0
	hasLocal := preloadLocalAuthEnabled

	if hasLocal {
		FocusInput("input[type=text]")
	}

	var doLogin func()
	doLogin = func() {
		if loading() {
			return
		}
		setErrMsg(ErrorInfo{})
		setLoading(true)

		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/auth/login", map[string]string{
				"username": username(),
				"password": password(),
			}, &resp)

			if err != nil {
				setErrMsg(ErrorInfo{Message: "Invalid credentials"})
				setLoading(false)
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
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

	onKeyDown := On{"keydown": func(evt *EventKeyboard) {
		if evt.Key() == "Enter" {
			evt.PreventDefault()
			doLogin()
		}
	}}

	// Build the form content
	var formContent []loom.Node

	// Demo mode hint
	if preloadDemoMode {
		formContent = append(formContent, Div(
			Apply(Attr{"class": "mb-4 px-4 py-3 rounded-lg border border-blue-500/30 bg-blue-500/10 text-sm text-blue-300"}),
			Span(Apply(Attr{"class": "font-semibold"}), Text("Demo Mode")),
			Span(Text(" — sign in with ")),
			Span(Apply(Attr{"class": "font-mono font-semibold"}), Text("admin")),
			Span(Text(" / ")),
			Span(Apply(Attr{"class": "font-mono font-semibold"}), Text("admin")),
		))
	}

	// OIDC provider buttons
	if hasOIDC {
		var oidcButtons []loom.Node
		for _, p := range preloadOIDCProviders {
			p := p
			label := fmt.Sprintf("Sign in with %s", p.Name)
			oidcButtons = append(oidcButtons, Button(
				Apply(Attr{"class": "w-full px-4 py-3 text-sm font-semibold rounded-lg border border-line-2 text-ink-1 bg-surface-1 hover:bg-surface-2 hover:border-line-3 transition-all duration-100 flex items-center justify-center gap-2"}),
				Apply(On{"click": func() {
					js.Global().Get("window").Get("location").Set("href", p.LoginURL)
				}}),
				Icon("key-round", 16),
				Text(label),
			))
		}
		formContent = append(formContent, Div(
			Apply(Attr{"class": "space-y-2"}),
			Fragment(oidcButtons...),
		))
	}

	// Divider between OIDC and local auth
	if hasOIDC && hasLocal {
		formContent = append(formContent, Div(
			Apply(Attr{"class": "flex items-center gap-3 my-6"}),
			Div(Apply(Attr{"class": "flex-1 h-px bg-line-1"})),
			Span(Apply(Attr{"class": "text-xs text-ink-3 uppercase tracking-wider"}), Text("or")),
			Div(Apply(Attr{"class": "flex-1 h-px bg-line-1"})),
		))
	}

	// Local auth form
	if hasLocal {
		formContent = append(formContent,
			Div(
				Apply(Attr{"class": "space-y-1"}),
				ErrorAlert(errMsg),
				FormField("Username", "text", "admin", username, func(v string) { setUsername(v) }),
				FormField("Password", "password", "", password, func(v string) { setPassword(v) }),
			),
			Button(
				Apply(Attr{"class": "w-full mt-6 px-4 py-3 text-sm font-semibold rounded-lg border border-wg-600/50 text-wg-400 bg-wg-600/5 hover:bg-wg-600/15 hover:border-wg-600/70 active:bg-wg-600/20 transition-all duration-100"}),
				Apply(On{"click": func() { doLogin() }}),
				Text("Sign In"),
			),
		)
	}

	// Subtitle text
	subtitle := "Enter your credentials to continue"
	if hasOIDC && !hasLocal {
		subtitle = "Use your SSO provider to continue"
	} else if hasOIDC && hasLocal {
		subtitle = "Choose a sign-in method"
	}

	return Div(
		Apply(Attr{"class": "flex min-h-screen bg-surface-0"}),

		// Left brand panel — hidden on mobile, dramatic on desktop
		Div(
			Apply(Attr{"class": "hidden md:flex w-[45%] bg-surface-1 border-r border-line-1 relative overflow-hidden flex-col items-center justify-center"}),
			// Subtle radial glow
			Div(Apply(Attr{"class": "absolute w-64 h-64 bg-wg-600/8 rounded-full blur-3xl"})),
			// Brand
			Div(
				Apply(Attr{"class": "relative z-10 text-center"}),
				Div(
					Apply(Attr{"class": "text-5xl font-extrabold tracking-tight"}),
					Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
					Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
				),
				P(Apply(Attr{"class": "text-ink-3 text-sm mt-4 tracking-wide"}), Text("WireGuard VPN Management")),
			),
			// Bottom accent gradient line
			Div(Apply(Attr{"class": "absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-wg-600/40 to-transparent"})),
		),

		// Right form panel
		Div(
			Apply(Attr{"class": "flex-1 flex items-center justify-center px-6"}),
			Div(
				Apply(Attr{"class": "w-full max-w-sm"}),
				Apply(onKeyDown),

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

				// Sign in heading
				Div(
					Apply(Attr{"class": "mb-8"}),
					H2(Apply(Attr{"class": "text-2xl font-bold text-ink-1 tracking-tight"}), Text("Sign in")),
					P(Apply(Attr{"class": "text-ink-3 text-sm mt-2"}), Text(subtitle)),
				),

				// Form content (OIDC buttons + divider + local form)
				Fragment(formContent...),
			),
		),
	)
}
