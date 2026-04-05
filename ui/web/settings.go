//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Package-level state for settings (survives refreshRoute re-mount)
var (
	settingsShowOIDCForm   bool
	settingsEditProviderID string
)

func SettingsView() loom.Node {
	settings, setSettings := Signal[*settingsData](nil)
	loading, setLoading := Signal(true)
	errMsg, setErrMsg := Signal(ErrorInfo{})

	loadSettings := func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/settings", nil, &resp); err != nil {
				setErrMsg(ErrorInfo{Message: "Failed to load settings"})
				setLoading(false)
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
				setLoading(false)
				return
			}
			var data settingsData
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				setErrMsg(ErrorInfo{Message: "Failed to parse settings"})
				setLoading(false)
				return
			}
			setSettings(&data)
			setLoading(false)
		}()
	}

	Effect(func() { loadSettings() })

	return Div(
		PageHeader("Settings", "Server configuration and SSO providers"),
		ErrorAlert(errMsg),
		LoadingView(loading),
		Show(func() bool { return !loading() && settings() != nil }, func() loom.Node {
			return Div(
				Apply(Attr{"class": "space-y-6"}),
				externalURLSection(settings),
				smtpSection(settings),
				oidcProvidersSection(settings),
			)
		}),
	)
}

func externalURLSection(settings func() *settingsData) loom.Node {
	url, setURL := Signal("")
	saving, setSaving := Signal(false)
	saved, setSaved := Signal(false)

	Effect(func() {
		s := settings()
		if s != nil {
			setURL(s.ExternalURL)
		}
	})

	doSave := func() {
		setSaving(true)
		setSaved(false)
		go func() {
			var resp apiResponse
			err := apiFetch("PUT", "/api/v1/settings", map[string]string{
				"external_url": url(),
			}, &resp)
			setSaving(false)
			if err == nil && resp.Error == "" {
				setSaved(true)
			}
		}()
	}

	return Card(
		CardHeader("General"),
		Div(
			Apply(Attr{"class": "space-y-4"}),
			Div(
				FormField("External URL", "text", "https://vpn.example.com", url, func(v string) { setURL(v) }),
				P(Apply(Attr{"class": "text-xs text-ink-3 mt-1"}), Text("Public URL for this server. Required for OIDC callback URLs.")),
			),
			Div(
				Apply(Attr{"class": "flex items-center gap-3"}),
				Btn("Save", "primary", doSave),
				Bind(func() loom.Node {
					msg := ""
					cls := "hidden"
					if saving() {
						msg = "Saving..."
						cls = "text-xs text-ink-3"
					} else if saved() {
						msg = "Saved"
						cls = "text-xs text-green-400"
					}
					return Span(Apply(Attr{"class": cls}), Text(msg))
				}),
			),
		),
	)
}

func oidcCallbackURL(settings func() *settingsData) string {
	s := settings()
	if s == nil || s.ExternalURL == "" {
		return "(set External URL above first)"
	}
	return s.ExternalURL + "/api/v1/auth/oidc/callback"
}

func copyableField(label, value string) loom.Node {
	copied, setCopied := Signal(false)

	return Div(
		Apply(Attr{"class": "mt-3"}),
		Div(Apply(Attr{"class": "text-xs font-medium text-ink-3 uppercase tracking-[0.08em] mb-1.5"}), Text(label)),
		Div(
			Apply(Attr{"class": "flex items-center gap-2"}),
			Div(
				Apply(Attr{"class": "flex-1 px-3 py-2 bg-surface-0 border border-line-1 rounded-lg text-xs text-ink-2 font-mono truncate select-all"}),
				Text(value),
			),
			Bind(func() loom.Node {
				btnLabel := "Copy"
				btnClass := "inline-flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border transition-all duration-100 cursor-pointer "
				if copied() {
					btnClass += "border-green-500/30 text-green-400 bg-green-500/5"
					btnLabel = "Copied"
				} else {
					btnClass += "border-line-2 text-ink-2 hover:text-ink-1 hover:bg-surface-2"
				}
				return Button(
					Apply(Attr{"class": btnClass}),
					Apply(On{"click": func() {
						if !copied() {
							js.Global().Get("navigator").Get("clipboard").Call("writeText", value)
							setCopied(true)
							js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
								setCopied(false)
								return nil
							}), 2000)
						}
					}}),
					Text(btnLabel),
				)
			}),
		),
	)
}

func oidcProvidersSection(settings func() *settingsData) loom.Node {
	cbURL := oidcCallbackURL(settings)

	return Div(
		Card(
			CardHeader("OIDC Providers",
				Btn("Add Provider", "primary", func() {
					settingsShowOIDCForm = !settingsShowOIDCForm
					settingsEditProviderID = ""
					refreshRoute()
				}),
			),

			// Callback URL info
			copyableField("Callback URL (register this in your OIDC provider)", cbURL),
			Div(Apply(Attr{"class": "mb-5"})),

			// Create/edit form
			func() loom.Node {
				if settingsShowOIDCForm {
					var editProvider *oidcProviderData
					if settingsEditProviderID != "" {
						s := settings()
						if s != nil {
							for i := range s.OIDCProviders {
								if s.OIDCProviders[i].ID == settingsEditProviderID {
									p := s.OIDCProviders[i]
									editProvider = &p
									break
								}
							}
						}
					}
					return Div(
						Apply(Attr{"class": "mb-6"}),
						oidcProviderForm(settingsEditProviderID, editProvider),
					)
				}
				return Span()
			}(),

			// Provider list
			func() loom.Node {
				s := settings()
				if s == nil || len(s.OIDCProviders) == 0 {
					return EmptyState("No OIDC providers configured")
				}

				cards := make([]loom.Node, 0, len(s.OIDCProviders))
				for _, p := range s.OIDCProviders {
					p := p
					enabledColor := "emerald"
					enabledLabel := "Enabled"
					if !p.Enabled {
						enabledColor = "red"
						enabledLabel = "Disabled"
					}

					cards = append(cards, Div(
						Apply(Attr{"class": "bg-surface-0 rounded-lg px-5 py-4 border border-line-1"}),
						Div(
							Apply(Attr{"class": "flex items-center justify-between"}),
							Div(
								Apply(Attr{"class": "min-w-0"}),
								Div(
									Apply(Attr{"class": "flex items-center gap-2"}),
									Span(Apply(Attr{"class": "text-sm font-semibold text-ink-1"}), Text(p.Name)),
									Badge(enabledLabel, enabledColor),
									Badge(p.DefaultRole, ""),
								),
								Div(Apply(Attr{"class": "text-xs text-ink-3 mt-1"}), Text(p.Issuer)),
								Div(Apply(Attr{"class": "text-xs text-ink-4 mt-0.5"}),
									Text(fmt.Sprintf("Client: %s", p.ClientID)),
								),
							),
							Div(
								Apply(Attr{"class": "flex items-center gap-0.5"}),
								IconBtn("settings", "Edit provider", func() {
									settingsShowOIDCForm = true
									settingsEditProviderID = p.ID
									refreshRoute()
								}),
								IconBtnDanger("trash-2", "Delete provider", func() {
									ConfirmAction(fmt.Sprintf("Delete OIDC provider %s?", p.Name), func() {
										go func() {
											apiFetch("DELETE", fmt.Sprintf("/api/v1/settings/oidc/%s", p.ID), nil, nil)
											settingsShowOIDCForm = false
											settingsEditProviderID = ""
											refreshRoute()
										}()
									})
								}),
							),
						),
					))
				}

				return Div(
					Apply(Attr{"class": "space-y-2"}),
					Fragment(cards...),
				)
			}(),
		),
	)
}

// Package-level state for SMTP settings form
var settingsShowSMTPForm bool

func smtpSection(settings func() *settingsData) loom.Node {
	return Card(
		CardHeader("Email (SMTP)",
			func() loom.Node {
				s := settings()
				if s != nil && s.SMTP != nil && s.SMTP.Host != "" {
					return Div(
						Apply(Attr{"class": "flex items-center gap-2"}),
						Btn("Edit", "ghost", func() {
							settingsShowSMTPForm = !settingsShowSMTPForm
							refreshRoute()
						}),
						Btn("Remove", "danger", func() {
							ConfirmAction("Remove SMTP configuration? Email features will be disabled.", func() {
								go func() {
									apiFetch("DELETE", "/api/v1/settings/smtp", nil, nil)
									settingsShowSMTPForm = false
									setSmtpEnabled(false)
									refreshRoute()
								}()
							})
						}),
					)
				}
				return Btn("Configure", "primary", func() {
					settingsShowSMTPForm = !settingsShowSMTPForm
					refreshRoute()
				})
			}(),
		),

		// Current config display
		func() loom.Node {
			s := settings()
			if s == nil || s.SMTP == nil || s.SMTP.Host == "" {
				if !settingsShowSMTPForm {
					return Div(
						Apply(Attr{"class": "text-sm text-ink-3 py-4 text-center"}),
						Text("No SMTP server configured. Email features are disabled."),
					)
				}
				return Span()
			}
			if settingsShowSMTPForm {
				return Span()
			}
			return Div(
				Apply(Attr{"class": "space-y-1 text-sm"}),
				smtpInfoRow("Host", fmt.Sprintf("%s:%s", s.SMTP.Host, s.SMTP.Port)),
				smtpInfoRow("From", s.SMTP.From),
				smtpInfoRow("Username", func() string {
					if s.SMTP.Username == "" {
						return "(none)"
					}
					return s.SMTP.Username
				}()),
				smtpInfoRow("Password", func() string {
					if s.SMTP.HasPassword {
						return "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"
					}
					return "(none)"
				}()),
				smtpInfoRow("TLS", s.SMTP.TLS),
				Div(
					Apply(Attr{"class": "pt-3"}),
					smtpTestButton(),
				),
			)
		}(),

		// Edit/create form
		func() loom.Node {
			if !settingsShowSMTPForm {
				return Span()
			}
			s := settings()
			var existing *smtpSettingsData
			if s != nil {
				existing = s.SMTP
			}
			return Div(
				Apply(Attr{"class": "mt-2"}),
				smtpSettingsForm(existing),
			)
		}(),
	)
}

func smtpInfoRow(label, value string) loom.Node {
	return Div(
		Apply(Attr{"class": "flex items-center gap-3 py-1"}),
		Span(Apply(Attr{"class": "w-20 text-xs font-medium text-ink-3 uppercase tracking-[0.08em]"}), Text(label)),
		Span(Apply(Attr{"class": "text-xs text-ink-2 font-mono"}), Text(value)),
	)
}

func smtpTestButton() loom.Node {
	testTo, setTestTo := Signal("")
	testing, setTesting := Signal(false)
	testResult, setTestResult := Signal("")
	showTestInput, setShowTestInput := Signal(false)

	doSend := func() {
		if testing() || testTo() == "" {
			return
		}
		setTesting(true)
		setTestResult("")
		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/settings/smtp/test", map[string]string{"to": testTo()}, &resp)
			setTesting(false)
			if err != nil {
				setTestResult(err.Error())
			} else {
				setTestResult("Test email sent!")
			}
		}()
	}

	return Div(
		Bind(func() loom.Node {
			show := showTestInput()
			isTesting := testing()
			resultMsg := testResult()

			// Toggle visibility via CSS, keeping identical DOM structure
			btnRowCls := "flex items-center gap-2"
			initBtnCls := "inline-flex items-center justify-center text-xs font-medium rounded-md transition-all duration-100 cursor-pointer px-3.5 py-2 text-ink-2 hover:text-ink-1 hover:bg-surface-2"
			if show {
				initBtnCls = "hidden"
				FocusInput(`input[placeholder="test@example.com"]`)
			} else {
				btnRowCls = "hidden"
			}

			resultCls := "hidden"
			resultIcon := ""
			if show && resultMsg != "" {
				if resultMsg == "Test email sent!" {
					resultCls = "inline-flex items-center gap-1.5 text-xs text-green-400 mt-2"
					resultIcon = icons["check"](14)
				} else {
					resultCls = "inline-flex items-center gap-1.5 text-xs text-red-400 mt-2"
					resultIcon = icons["triangle-alert"](14)
				}
			}

			sendBtnLabel := "Send"
			sendBtnCls := "inline-flex items-center justify-center text-xs font-medium rounded-md transition-all duration-100 cursor-pointer px-3 py-1.5 border border-wg-500/40 text-wg-400 bg-wg-600/10 hover:bg-wg-600/20"
			if isTesting {
				sendBtnLabel = "Sending..."
				sendBtnCls = "inline-flex items-center justify-center text-xs font-medium rounded-md px-3 py-1.5 border border-line-1 text-ink-4 bg-surface-3 cursor-not-allowed"
			}

			return Div(
				Button(
					Apply(Attr{"class": initBtnCls}),
					Apply(On{"click": func() {
						setShowTestInput(true)
					}}),
					Text("Send Test Email"),
				),
				Div(
					Apply(Attr{"class": btnRowCls}),
					Input(
						Apply(Attr{
							"class":       "flex-1 px-3 py-1.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/40 transition-colors font-mono",
							"type":        "email",
							"placeholder": "test@example.com",
						}),
						Apply(On{"input": func(evt *EventInput) {
							setTestTo(evt.InputValue())
						}}),
						Apply(On{"keydown": func(evt *EventKeyboard) {
							if evt.Key() == "Enter" {
								evt.PreventDefault()
								doSend()
							}
						}}),
					),
					Button(
						Apply(Attr{"class": sendBtnCls}),
						Apply(On{"click": func() { doSend() }}),
						Text(sendBtnLabel),
					),
					Button(
						Apply(Attr{"class": "text-xs text-ink-3 hover:text-ink-1 px-2 py-1.5"}),
						Apply(On{"click": func() {
							setShowTestInput(false)
							setTestResult("")
						}}),
						Text("Cancel"),
					),
				),
				Span(
					Apply(Attr{"class": resultCls}),
					Span(Apply(innerHTML(resultIcon))),
					Span(Text(resultMsg)),
				),
			)
		}),
	)
}

func smtpSettingsForm(existing *smtpSettingsData) loom.Node {
	initHost := ""
	initPort := "587"
	initUsername := ""
	initFrom := ""
	initTLS := "starttls"

	if existing != nil && existing.Host != "" {
		initHost = existing.Host
		initPort = existing.Port
		initUsername = existing.Username
		initFrom = existing.From
		initTLS = existing.TLS
	}

	host, setHost := Signal(initHost)
	port, setPort := Signal(initPort)
	username, setUsername := Signal(initUsername)
	password, setPassword := Signal("")
	from, setFrom := Signal(initFrom)
	tlsMode, setTLSMode := Signal(initTLS)
	errMsg, setErrMsg := Signal(ErrorInfo{})

	isEdit := existing != nil && existing.Host != ""

	FocusInput(`input[placeholder="smtp.example.com"]`)

	doSave := func() {
		setErrMsg(ErrorInfo{})
		if host() == "" {
			setErrMsg(ErrorInfo{Message: "Host is required"})
			return
		}
		if from() == "" {
			setErrMsg(ErrorInfo{Message: "From address is required"})
			return
		}
		if !isEdit && password() == "" && username() != "" {
			setErrMsg(ErrorInfo{Message: "Password is required for authenticated SMTP"})
			return
		}

		body := map[string]string{
			"host":     host(),
			"port":     port(),
			"username": username(),
			"password": password(),
			"from":     from(),
			"tls":      tlsMode(),
		}

		go func() {
			var resp apiResponse
			err := apiFetch("PUT", "/api/v1/settings/smtp", body, &resp)
			if err != nil {
				setErrMsg(apiErrorInfo(err))
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
				return
			}
			settingsShowSMTPForm = false
			setSmtpEnabled(true)
			showToast("SMTP settings saved")
			refreshRoute()
		}()
	}

	return Div(
		Apply(Attr{"class": "bg-surface-0 rounded-lg p-5 border border-line-1"}),
		Div(
			Apply(Attr{"class": "text-sm font-semibold text-ink-1 mb-4"}),
			Text(func() string {
				if isEdit {
					return "Edit SMTP Settings"
				}
				return "Configure SMTP"
			}()),
		),
		ErrorAlert(errMsg),
		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Host", "text", "smtp.example.com", host, func(v string) { setHost(v) }),
			FormField("Port", "text", "587", port, func(v string) { setPort(v) }),
			FormField("From Address", "email", "wgrift@example.com", from, func(v string) { setFrom(v) }),
			Div(
				Apply(Attr{"class": "mb-4"}),
				Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("TLS Mode")),
				Elem("select",
					Apply(Attr{
						"class": "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm focus:outline-none focus:border-wg-600/40 focus:ring-1 focus:ring-wg-600/15 transition-colors",
					}),
					Apply(On{"change": func(evt *EventInput) {
						setTLSMode(evt.InputValue())
					}}),
					Elem("option", Apply(Attr{"value": "starttls"}), Text("STARTTLS (port 587)")),
					Elem("option", Apply(Attr{"value": "tls"}), Text("TLS (port 465)")),
					Elem("option", Apply(Attr{"value": "none"}), Text("None (port 25)")),
				),
			),
			FormField("Username", "text", "username (optional)", username, func(v string) { setUsername(v) }),
			func() loom.Node {
				placeholder := "password"
				if isEdit {
					placeholder = "leave blank to keep current"
				}
				return FormField("Password", "password", placeholder, password, func(v string) { setPassword(v) })
			}(),
		),
		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Save", "primary", doSave),
			Btn("Cancel", "ghost", func() {
				settingsShowSMTPForm = false
				refreshRoute()
			}),
		),
	)
}

func oidcProviderForm(editID string, existing *oidcProviderData) loom.Node {
	initName := ""
	initIssuer := ""
	initClientID := ""
	initScopes := "openid profile email groups"
	initAdminClaim := ""
	initAdminValue := ""
	initDefaultRole := "viewer"
	initEnabled := true

	isEdit := editID != ""

	if isEdit && existing != nil {
		initName = existing.Name
		initIssuer = existing.Issuer
		initClientID = existing.ClientID
		initScopes = existing.Scopes
		initAdminClaim = existing.AdminClaim
		initAdminValue = existing.AdminValue
		initDefaultRole = existing.DefaultRole
		initEnabled = existing.Enabled
	}

	name, setName := Signal(initName)
	issuer, setIssuer := Signal(initIssuer)
	clientID, setClientID := Signal(initClientID)
	clientSecret, setClientSecret := Signal("")
	scopes, setScopes := Signal(initScopes)
	adminClaim, setAdminClaim := Signal(initAdminClaim)
	adminValue, setAdminValue := Signal(initAdminValue)
	defaultRole, setDefaultRole := Signal(initDefaultRole)
	enabled, setEnabled := Signal(initEnabled)
	errMsg, setErrMsg := Signal(ErrorInfo{})

	FocusInput(`input[placeholder="Pocket ID"]`)

	doSave := func() {
		setErrMsg(ErrorInfo{})
		if name() == "" || issuer() == "" || clientID() == "" {
			setErrMsg(ErrorInfo{Message: "Name, Issuer URL, and Client ID are required"})
			return
		}
		if !isEdit && clientSecret() == "" {
			setErrMsg(ErrorInfo{Message: "Client Secret is required"})
			return
		}

		body := map[string]any{
			"name":          name(),
			"issuer":        issuer(),
			"client_id":     clientID(),
			"client_secret": clientSecret(),
			"scopes":        scopes(),
			"auto_discover": true,
			"admin_claim":   adminClaim(),
			"admin_value":   adminValue(),
			"default_role":  defaultRole(),
			"enabled":       enabled(),
		}

		go func() {
			var resp apiResponse
			var err error
			if isEdit {
				err = apiFetch("PUT", fmt.Sprintf("/api/v1/settings/oidc/%s", editID), body, &resp)
			} else {
				err = apiFetch("POST", "/api/v1/settings/oidc", body, &resp)
			}
			if err != nil {
				setErrMsg(apiErrorInfo(err))
				return
			}
			if resp.Error != "" {
				setErrMsg(ErrorInfo{Message: resp.Error})
				return
			}
			settingsShowOIDCForm = false
			settingsEditProviderID = ""
			refreshRoute()
		}()
	}

	title := "New OIDC Provider"
	if isEdit {
		title = "Edit OIDC Provider"
	}

	return Div(
		Apply(Attr{"class": "bg-surface-0 rounded-lg p-5 border border-line-1"}),
		Div(
			Apply(Attr{"class": "text-sm font-semibold text-ink-1 mb-4"}),
			Text(title),
		),
		ErrorAlert(errMsg),
		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Name", "text", "Pocket ID", name, func(v string) { setName(v) }),
			FormField("Issuer URL", "text", "https://id.example.com", issuer, func(v string) { setIssuer(v) }),
			FormField("Client ID", "text", "wgrift", clientID, func(v string) { setClientID(v) }),
			func() loom.Node {
				placeholder := "client secret"
				if isEdit {
					placeholder = "leave blank to keep current"
				}
				return FormField("Client Secret", "password", placeholder, clientSecret, func(v string) { setClientSecret(v) })
			}(),
			FormField("Scopes", "text", "openid profile email groups", scopes, func(v string) { setScopes(v) }),
			Div(
				Apply(Attr{"class": "mb-4"}),
				Elem("label", Apply(Attr{"class": "block text-xs font-medium text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("Default Role")),
				Elem("select",
					Apply(Attr{
						"class": "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-lg text-ink-1 text-sm focus:outline-none focus:border-wg-600/50 focus:ring-1 focus:ring-wg-600/20 transition-colors",
					}),
					Apply(On{"change": func(evt *EventInput) {
						setDefaultRole(evt.InputValue())
					}}),
					Elem("option", Apply(Attr{"value": "viewer"}), Text("Viewer")),
					Elem("option", Apply(Attr{"value": "admin"}), Text("Admin")),
				),
			),
			FormField("Admin Claim", "text", "groups", adminClaim, func(v string) { setAdminClaim(v) }),
			FormField("Admin Value", "text", "admins", adminValue, func(v string) { setAdminValue(v) }),
		),
		Div(
			Apply(Attr{"class": "flex items-center gap-3 mt-2 mb-4"}),
			Elem("label",
				Apply(Attr{"class": "flex items-center gap-2 text-sm text-ink-1 cursor-pointer"}),
				Bind(func() loom.Node {
					attrs := Attr{
						"type":  "checkbox",
						"class": "rounded border-line-2 bg-surface-0 text-wg-500 focus:ring-wg-600/30",
					}
					if enabled() {
						attrs["checked"] = "true"
					}
					return Elem("input",
						Apply(attrs),
						Apply(On{"change": func() { setEnabled(!enabled()) }}),
					)
				}),
				Text("Enabled"),
			),
		),
		Div(
			Apply(Attr{"class": "flex gap-2"}),
			Btn(func() string {
				if isEdit {
					return "Update Provider"
				}
				return "Create Provider"
			}(), "primary", doSave),
			Btn("Cancel", "ghost", func() {
				settingsShowOIDCForm = false
				settingsEditProviderID = ""
				refreshRoute()
			}),
		),
	)
}
