//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Package-level state for user management (survives refreshRoute re-mount)
var (
	usersShowForm       bool   // show create user form
	usersPasswordUserID string // show change password form for this user ID
)

func UsersView() loom.Node {
	users, setUsers := Signal[[]userData](nil)
	loading, setLoading := Signal(true)

	loadUsers := func() {
		go func() {
			var resp apiResponse
			if err := apiFetch("GET", "/api/v1/users", nil, &resp); err != nil {
				setLoading(false)
				return
			}
			var list []userData
			json.Unmarshal(resp.Data, &list)
			setUsers(list)
			setLoading(false)
		}()
	}

	Effect(func() { loadUsers() })

	return Div(
		PageHeader("Users", "Manage access to wgRift",
			Btn("Add User", "primary", func() {
				usersShowForm = !usersShowForm
				usersPasswordUserID = ""
				refreshRoute()
			}),
		),

		// Create user form
		func() loom.Node {
			if usersShowForm {
				return Div(
					Apply(Attr{"class": "mb-6"}),
					createUserForm(),
				)
			}
			return Span()
		}(),

		LoadingView(loading),
		Show(func() bool { return !loading() }, func() loom.Node {
			list := users()
			if len(list) == 0 {
				return EmptyState("No users")
			}

			cards := make([]loom.Node, 0, len(list))
			for _, u := range list {
				u := u

				// Avatar initial
				initial := strings.ToUpper(u.Username[:1])

				// Role badge color
				roleColor := ""
				if u.Role == "admin" {
					roleColor = "teal"
				}

				// Action buttons
				var passwordBtn loom.Node
				if u.OIDCProvider == "" {
					passwordBtn = IconBtn("settings", "Change password", func() {
						usersShowForm = false
						if usersPasswordUserID == u.ID {
							usersPasswordUserID = ""
						} else {
							usersPasswordUserID = u.ID
						}
						refreshRoute()
					})
				} else if u.OIDCIssuer != "" {
					issuerURL := u.OIDCIssuer
					passwordBtn = Elem("a",
						Apply(Attr{
							"class":  "w-8 h-8 rounded-md flex items-center justify-center text-ink-4 hover:text-ink-1 hover:bg-surface-2 transition-all duration-100",
							"title":  "Open identity provider",
							"href":   issuerURL,
							"target": "_blank",
							"rel":    "noopener noreferrer",
						}),
						Icon("external-link", 15),
					)
				} else {
					passwordBtn = Span()
				}

				userActions := Div(
					Apply(Attr{"class": "flex items-center gap-0.5"}),
					passwordBtn,
					func() loom.Node {
						if currentUser() != nil && currentUser().ID == u.ID {
							return Span()
						}
						deleteLabel := fmt.Sprintf("Delete user %s? This cannot be undone.", u.Username)
						if u.OIDCProvider != "" {
							deleteLabel = fmt.Sprintf("Remove SSO user %s? They will be re-created on next SSO login.", u.Username)
						}
						return IconBtnDanger("trash-2", "Delete user", func() {
							ConfirmAction(deleteLabel, func() {
								go func() {
									apiFetch("DELETE", fmt.Sprintf("/api/v1/users/%s", u.ID), nil, nil)
									usersPasswordUserID = ""
									refreshRoute()
								}()
							})
						})
					}(),
				)

				// Build card children
				cardChildren := []loom.Node{
					Apply(Attr{"class": "bg-surface-1 rounded-lg px-6 py-4"}),
					Div(
						Apply(Attr{"class": "flex items-center justify-between"}),
						// Left: avatar + name + role
						Div(
							Apply(Attr{"class": "flex items-center gap-4 min-w-0"}),
							// Avatar circle with initial
							Div(
								Apply(Attr{"class": "w-10 h-10 rounded-lg bg-wg-600/15 flex items-center justify-center text-sm font-bold text-wg-400 uppercase flex-shrink-0"}),
								Text(initial),
							),
							Div(
								Apply(Attr{"class": "min-w-0"}),
								Div(
									Apply(Attr{"class": "flex items-center gap-2"}),
									Span(Apply(Attr{"class": "text-sm font-semibold text-ink-1"}), Text(u.Username)),
									Badge(u.Role, roleColor),
									func() loom.Node {
										if u.OIDCProvider != "" {
											return Badge("SSO: "+u.OIDCProvider, "amber")
										}
										return Span()
									}(),
								),
								func() loom.Node {
									if u.DisplayName != "" {
										return Div(Apply(Attr{"class": "text-xs text-ink-3 mt-0.5"}), Text(u.DisplayName))
									}
									return Span()
								}(),
							),
						),
						// Right: actions
						userActions,
					),
				}

				// Inline password form if active
				if usersPasswordUserID == u.ID {
					cardChildren = append(cardChildren, Div(
						Apply(Attr{"class": "mt-4 pt-4 border-t border-line-1"}),
						changePasswordForm(u.ID, u.Username),
					))
				}

				cards = append(cards, Div(cardChildren...))
			}

			return Div(
				Apply(Attr{"class": "space-y-2"}),
				Fragment(cards...),
			)
		}),
	)
}

func createUserForm() loom.Node {
	username, setUsername := Signal("")
	password, setPassword := Signal("")
	displayName, setDisplayName := Signal("")
	role, setRole := Signal("viewer")
	errMsg, setErrMsg := Signal("")
	FocusInput(`input[placeholder="jdoe"]`)

	doCreate := func() {
		setErrMsg("")
		if username() == "" {
			setErrMsg("Username is required")
			return
		}
		if password() == "" {
			setErrMsg("Password is required")
			return
		}
		if len(password()) < 16 {
			setErrMsg("Password must be at least 16 characters")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("POST", "/api/v1/users", map[string]any{
				"username":     username(),
				"password":     password(),
				"display_name": displayName(),
				"role":         role(),
			}, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			usersShowForm = false
			refreshRoute()
		}()
	}

	return Card(
		CardHeader("New User"),
		ErrorAlert(errMsg),
		Div(
			Apply(Attr{"class": "grid grid-cols-1 sm:grid-cols-2 gap-4"}),
			FormField("Username", "text", "jdoe", username, func(v string) { setUsername(v) }),
			FormField("Display Name", "text", "John Doe", displayName, func(v string) { setDisplayName(v) }),
			FormField("Password", "password", "min 16 characters", password, func(v string) { setPassword(v) }),
			Div(
				Apply(Attr{"class": "mb-4"}),
				Elem("label", Apply(Attr{"class": "block text-xs font-medium text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("Role")),
				Elem("select",
					Apply(Attr{
						"class": "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-lg text-ink-1 text-sm focus:outline-none focus:border-wg-600/50 focus:ring-1 focus:ring-wg-600/20 transition-colors",
					}),
					Apply(On{"change": func(evt *EventInput) {
						setRole(evt.InputValue())
					}}),
					Elem("option", Apply(Attr{"value": "viewer"}), Text("Viewer")),
					Elem("option", Apply(Attr{"value": "admin"}), Text("Admin")),
				),
			),
		),
		Div(
			Apply(Attr{"class": "flex gap-2 mt-2"}),
			Btn("Create User", "primary", doCreate),
			Btn("Cancel", "ghost", func() {
				usersShowForm = false
				refreshRoute()
			}),
		),
	)
}

func changePasswordForm(userID, username string) loom.Node {
	password, setPassword := Signal("")
	errMsg, setErrMsg := Signal("")
	success, setSuccess := Signal(false)
	FocusInput(`input[placeholder="min 16 characters"]`)

	doChange := func() {
		setErrMsg("")
		setSuccess(false)
		if password() == "" {
			setErrMsg("Password is required")
			return
		}
		if len(password()) < 16 {
			setErrMsg("Password must be at least 16 characters")
			return
		}
		go func() {
			var resp apiResponse
			err := apiFetch("PUT", fmt.Sprintf("/api/v1/users/%s/password", userID), map[string]string{
				"password": password(),
			}, &resp)
			if err != nil {
				setErrMsg(err.Error())
				return
			}
			if resp.Error != "" {
				setErrMsg(resp.Error)
				return
			}
			setSuccess(true)
			setPassword("")
		}()
	}

	return Div(
		Div(
			Apply(Attr{"class": "text-sm font-medium text-ink-1 mb-3"}),
			Text(fmt.Sprintf("Change password for %s", username)),
		),
		ErrorAlert(errMsg),
		Bind(func() loom.Node {
			if success() {
				return Div(
					Apply(Attr{"class": "mb-3 p-3 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm"}),
					Text("Password updated successfully"),
				)
			}
			return Div(Apply(Attr{"class": "hidden"}), Text(""))
		}),
		Div(
			Apply(Attr{"class": "flex items-end gap-3"}),
			Div(
				Apply(Attr{"class": "flex-1"}),
				FormField("New Password", "password", "min 16 characters", password, func(v string) { setPassword(v) }),
			),
			Div(
				Apply(Attr{"class": "flex gap-2 pb-4"}),
				Btn("Update", "primary", doChange),
				Btn("Close", "ghost", func() {
					usersPasswordUserID = ""
					refreshRoute()
				}),
			),
		),
	)
}
