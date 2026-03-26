//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

func LoginView() loom.Node {
	username, setUsername := Signal("")
	password, setPassword := Signal("")
	errMsg, setErrMsg := Signal("")
	loading, setLoading := Signal(false)

	OnRenderSettled(func() {
		if el := js.Global().Get("document").Call("querySelector", "input[type=text]"); el.Truthy() {
			el.Call("focus")
		}
	})

	var doLogin func()
	doLogin = func() {
		if loading() {
			return
		}
		setErrMsg("")
		setLoading(true)

		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/auth/login", map[string]string{
				"username": username(),
				"password": password(),
			}, &resp)

			if err != nil {
				setErrMsg("Invalid credentials")
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
				// Reload page — session cookie is set, checkSession will authenticate
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

	return Div(
		Apply(Attr{"class": "flex items-center justify-center min-h-screen bg-gray-50"}),
		Div(
			Apply(Attr{"class": "w-full max-w-sm"}),
			Apply(onKeyDown),

			// Logo
			Div(
				Apply(Attr{"class": "text-center mb-8"}),
				Div(
					Apply(Attr{"class": "inline-flex items-center gap-1 text-3xl font-bold"}),
					Span(Apply(Attr{"class": "text-teal-400"}), Text("wg")),
					Span(Apply(Attr{"class": "text-gray-700"}), Text("Rift")),
				),
				P(Apply(Attr{"class": "text-gray-400 text-sm mt-2"}), Text("WireGuard VPN Management")),
			),

			// Form card
			Div(
				Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-6"}),

				ErrorAlert(errMsg),

				FormField("Username", "text", "admin", username, func(v string) { setUsername(v) }),
				FormField("Password", "password", "", password, func(v string) { setPassword(v) }),

				Button(
					Apply(Attr{"class": "w-full px-4 py-2.5 text-sm font-medium rounded-md bg-teal-600 border border-teal-600 text-white hover:bg-teal-700 transition-colors"}),
					Apply(On{"click": func() { doLogin() }}),
					Text("Sign In"),
				),
			),
		),
	)
}
