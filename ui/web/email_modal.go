//go:build js && wasm

package main

import (
	"fmt"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

type emailModalState struct {
	PeerName    string
	InterfaceID string
	PeerID      string
}

var (
	emailState    func() *emailModalState
	setEmailState func(*emailModalState)
)

func initEmailModal() {
	emailState, setEmailState = Signal[*emailModalState](nil)
}

// ShowEmailModal opens the email modal for a peer.
func ShowEmailModal(peerName, ifaceID, peerID string) {
	setEmailState(&emailModalState{
		PeerName:    peerName,
		InterfaceID: ifaceID,
		PeerID:      peerID,
	})
}

func dismissEmailModal() {
	setEmailState(nil)
}

// EmailModal renders the email configuration modal overlay.
func EmailModal() loom.Node {
	emailTo, setEmailTo := Signal("")
	emailNote, setEmailNote := Signal("")
	sending, setSending := Signal(false)
	emailErr, setEmailErr := Signal(ErrorInfo{})

	return Bind(func() loom.Node {
		st := emailState()
		visible := st != nil
		peerName := ""
		if visible {
			peerName = st.PeerName
			if peerName == "" {
				peerName = st.PeerID[:8] + "..."
			}
		}

		backdropClass := "fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in"
		cardClass := "bg-surface-2 border border-line-1 rounded-lg max-w-md w-full mx-4 overflow-hidden animate-scale-in"
		if !visible {
			backdropClass = "fixed inset-0 z-50 hidden"
			cardClass = "bg-surface-2 border border-line-1 rounded-lg max-w-md w-full mx-4 overflow-hidden hidden"
		}

		isSending := sending()
		btnLabel := "Send"
		btnClass := "inline-flex items-center justify-center gap-1.5 text-xs font-semibold rounded-md transition-all duration-100 cursor-pointer px-4 py-2 border border-wg-500/40 text-wg-400 bg-wg-600/10 hover:bg-wg-600/20 hover:border-wg-500/60"
		if isSending {
			btnLabel = "Sending..."
			btnClass = "inline-flex items-center justify-center gap-1.5 text-xs font-semibold rounded-md px-4 py-2 border border-line-1 text-ink-4 bg-surface-3 cursor-not-allowed"
		}

		return Div(
			Apply(Attr{"class": backdropClass}),
			Apply(On{"click": func() {
				if !sending() {
					dismissEmailModal()
				}
			}}),
			Div(
				Apply(Attr{"class": cardClass}),
				Apply(On{"click": func(evt *Event) { evt.StopPropagation() }}),
				// Status stripe
				Div(Apply(Attr{"class": "h-[2px] bg-wg-500"})),
				Div(
					Apply(Attr{"class": "px-6 pt-5 pb-0"}),
					Div(
						Apply(Attr{"class": "flex items-start gap-4"}),
						Div(
							Apply(Attr{"class": "flex-shrink-0 w-9 h-9 rounded-md bg-wg-600/10 flex items-center justify-center"}),
							Span(Apply(Attr{"class": "text-wg-400"}), Apply(innerHTML(icons["mail"](16)))),
						),
						Div(
							Apply(Attr{"class": "flex-1 min-w-0"}),
							P(Apply(Attr{"class": "text-sm font-semibold text-ink-1"}), Text("Email Configuration")),
							P(Apply(Attr{"class": "mt-0.5 text-xs text-ink-3"}), Text(peerName)),
						),
					),
				),
				Div(
					Apply(Attr{"class": "px-6 pt-4 pb-0"}),
					ErrorAlert(func() ErrorInfo { return emailErr() }),
					Div(
						Apply(Attr{"class": "mb-4"}),
						Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("Recipient Email")),
						Input(
							Apply(Attr{
								"class":       "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/40 focus:ring-1 focus:ring-wg-600/15 transition-colors font-mono",
								"type":        "email",
								"placeholder": "user@example.com",
								"id":          "email-modal-to",
							}),
							Apply(On{"input": func(evt *EventInput) {
								setEmailTo(evt.InputValue())
							}}),
						),
					),
					Div(
						Apply(Attr{"class": "mb-4"}),
						Elem("label", Apply(Attr{"class": "block text-[11px] font-semibold text-ink-3 mb-2 uppercase tracking-[0.08em]"}), Text("Note (optional)")),
						Elem("textarea",
							Apply(Attr{
								"class":       "w-full px-3.5 py-2.5 bg-surface-0 border border-line-1 rounded-md text-ink-1 text-sm placeholder-ink-4 focus:outline-none focus:border-wg-600/40 focus:ring-1 focus:ring-wg-600/15 transition-colors",
								"placeholder": "Here's your VPN configuration...",
								"rows":        "3",
							}),
							Apply(On{"input": func(evt *EventInput) {
								setEmailNote(evt.InputValue())
							}}),
						),
					),
				),
				Div(
					Apply(Attr{"class": "flex items-center justify-end gap-3 px-6 py-4"}),
					Button(
						Apply(Attr{"class": "inline-flex items-center justify-center text-xs font-medium rounded-md transition-all duration-100 cursor-pointer px-4 py-2 text-ink-3 hover:bg-surface-3 hover:text-ink-1"}),
						Apply(On{"click": func() {
							if !sending() {
								setEmailTo("")
								setEmailNote("")
								setEmailErr(ErrorInfo{})
								dismissEmailModal()
							}
						}}),
						Text("Cancel"),
					),
					Button(
						Apply(Attr{"class": btnClass}),
						Apply(On{"click": func() {
							if sending() {
								return
							}
							to := emailTo()
							if to == "" {
								setEmailErr(ErrorInfo{Message: "Email address is required"})
								return
							}
							st := emailState()
							if st == nil {
								return
							}

							setSending(true)
							setEmailErr(ErrorInfo{})

							go func() {
								var resp apiResponse
								err := apiFetch("POST",
									fmt.Sprintf("/api/v1/interfaces/%s/peers/%s/email", st.InterfaceID, st.PeerID),
									map[string]string{"to": to, "note": emailNote()},
									&resp,
								)
								setSending(false)

								if err != nil {
									setEmailErr(apiErrorInfo(err))
									return
								}

								sentTo := to
								setEmailTo("")
								setEmailNote("")
								setEmailErr(ErrorInfo{})
								dismissEmailModal()
								showToast("Config sent to " + sentTo)
							}()
						}}),
						Span(Apply(innerHTML(icons["send"](14)))),
						Span(Text(btnLabel)),
					),
				),
			),
		)
	})
}
