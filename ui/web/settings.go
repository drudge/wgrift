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
