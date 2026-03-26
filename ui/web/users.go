//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"

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
		Div(
			Apply(Attr{"class": "flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-8"}),
			H2(Apply(Attr{"class": "text-xl font-semibold text-gray-900"}), Text("Users")),
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
			rows := make([]loom.Node, 0, len(list)*2)
			for _, u := range list {
				u := u
				roleBadge := Badge(u.Role, func() string {
					if u.Role == "admin" {
						return "teal"
					}
					return ""
				}())
				userActions := Div(
					Apply(Attr{"class": "flex items-center gap-0.5"}),
					IconBtn("settings", "Change password", func() {
						usersShowForm = false
						if usersPasswordUserID == u.ID {
							usersPasswordUserID = ""
						} else {
							usersPasswordUserID = u.ID
						}
						refreshRoute()
					}),
					func() loom.Node {
						if currentUser() != nil && currentUser().ID == u.ID {
							return Span()
						}
						return IconBtnDanger("trash-2", "Delete user", func() {
							ConfirmAction(fmt.Sprintf("Delete user %s? This cannot be undone.", u.Username), func() {
								go func() {
									apiFetch("DELETE", fmt.Sprintf("/api/v1/users/%s", u.ID), nil, nil)
									usersPasswordUserID = ""
									refreshRoute()
								}()
							})
						})
					}(),
				)

				// Mobile card
				cardChildren := []loom.Node{
					Apply(Attr{"class": "bg-white border border-gray-200 rounded-lg p-4"}),
					Div(
						Apply(Attr{"class": "flex items-center justify-between"}),
						Div(
							Apply(Attr{"class": "min-w-0"}),
							Div(
								Apply(Attr{"class": "flex items-center gap-2 mb-0.5"}),
								Span(Apply(Attr{"class": "text-sm font-medium text-gray-900"}), Text(u.Username)),
								roleBadge,
							),
							func() loom.Node {
								if u.DisplayName != "" {
									return Div(Apply(Attr{"class": "text-xs text-gray-400"}), Text(u.DisplayName))
								}
								return Span()
							}(),
						),
						userActions,
					),
				}
				if usersPasswordUserID == u.ID {
					cardChildren = append(cardChildren, Div(
						Apply(Attr{"class": "mt-3 pt-3 border-t border-gray-100"}),
						changePasswordForm(u.ID, u.Username),
					))
				}
				cards = append(cards, Div(cardChildren...))

				// Desktop row
				rows = append(rows, Elem("tr",
					Apply(Attr{"class": "border-b border-gray-100 hover:bg-gray-50 transition-colors"}),
					Elem("td", Apply(Attr{"class": "px-4 py-3 text-sm font-medium text-gray-900"}), Text(u.Username)),
					Elem("td", Apply(Attr{"class": "px-4 py-3 text-sm text-gray-500"}), Text(u.DisplayName)),
					Elem("td", Apply(Attr{"class": "px-4 py-3"}), Badge(u.Role, func() string {
						if u.Role == "admin" {
							return "teal"
						}
						return ""
					}())),
					Elem("td", Apply(Attr{"class": "px-4 py-3"}),
						Div(
							Apply(Attr{"class": "flex items-center gap-0.5 justify-end"}),
							IconBtn("settings", "Change password", func() {
								usersShowForm = false
								if usersPasswordUserID == u.ID {
									usersPasswordUserID = ""
								} else {
									usersPasswordUserID = u.ID
								}
								refreshRoute()
							}),
							func() loom.Node {
								if currentUser() != nil && currentUser().ID == u.ID {
									return Span()
								}
								return IconBtnDanger("trash-2", "Delete user", func() {
									ConfirmAction(fmt.Sprintf("Delete user %s? This cannot be undone.", u.Username), func() {
										go func() {
											apiFetch("DELETE", fmt.Sprintf("/api/v1/users/%s", u.ID), nil, nil)
											usersPasswordUserID = ""
											refreshRoute()
										}()
									})
								})
							}(),
						),
					),
				))
				if usersPasswordUserID == u.ID {
					rows = append(rows, Elem("tr",
						Apply(Attr{"class": "border-b border-gray-100 bg-gray-50"}),
						Elem("td", Apply(Attr{"class": "p-4", "colspan": "4"}),
							changePasswordForm(u.ID, u.Username),
						),
					))
				}
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
								Apply(Attr{"class": "border-b border-gray-200 text-left text-[11px] uppercase tracking-widest text-gray-400"}),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Username")),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Display Name")),
								Elem("th", Apply(Attr{"class": "px-4 py-3"}), Text("Role")),
								Elem("th", Apply(Attr{"class": "px-4 py-3 text-right"}), Text("Actions")),
							),
						),
						Elem("tbody", rows...),
					),
				),
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
				Elem("label", Apply(Attr{"class": "block text-sm text-gray-600 mb-1.5"}), Text("Role")),
				Elem("select",
					Apply(Attr{
						"class": "w-full px-3 py-2 bg-white border border-gray-300 rounded-md text-gray-900 text-sm focus:outline-none focus:border-teal-500 focus:ring-1 focus:ring-teal-500/20",
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
			Apply(Attr{"class": "text-sm font-medium text-gray-700 mb-3"}),
			Text(fmt.Sprintf("Change password for %s", username)),
		),
		ErrorAlert(errMsg),
		Bind(func() loom.Node {
			if success() {
				return Div(
					Apply(Attr{"class": "mb-3 p-3 bg-emerald-50 border border-emerald-200 rounded-md text-emerald-700 text-sm"}),
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
