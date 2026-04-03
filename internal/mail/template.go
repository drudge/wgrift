package mail

import (
	"bytes"
	"fmt"
	"html/template"
)

var peerConfigTmpl = template.Must(template.New("peer_config").Parse(`<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark">
<meta name="supported-color-schemes" content="light dark">
<!--[if mso]>
<noscript>
<xml>
<o:OfficeDocumentSettings>
<o:PixelsPerInch>96</o:PixelsPerInch>
</o:OfficeDocumentSettings>
</xml>
</noscript>
<style>
td,th,div,p,a,h1,h2,h3,h4,h5,h6 {font-family:"Segoe UI",Helvetica,Arial,sans-serif;}
table {border-collapse:collapse;}
</style>
<![endif]-->
<style>
:root { color-scheme: light dark; }
body, .body-bg { background-color: #f2f2f7 !important; }
.card-bg { background-color: #ffffff !important; }
.card-border { border: 1px solid #e5e5ea !important; }
.header-border { border-bottom: 1px solid #e5e5ea !important; }
.footer-border { border-top: 1px solid #e5e5ea !important; }
.h1-text { color: #1c1c1e !important; }
.h2-text { color: #1c1c1e !important; }
.sub-text { color: #8e8e93 !important; }
.body-text { color: #3a3a3c !important; }
.muted-text { color: #8e8e93 !important; }
.note-bg { background-color: #f2f2f7 !important; }
.note-border { border: 1px solid #e5e5ea !important; }
.note-text { color: #3a3a3c !important; }
.step-bg { background-color: #fee2e2 !important; }
.step-text { color: #dc2626 !important; }
.link { color: #dc2626 !important; }
.attach-bg { background-color: #f2f2f7 !important; }
.attach-border { border: 1px solid #e5e5ea !important; }
.attach-text { color: #8e8e93 !important; }
.attach-name { color: #3a3a3c !important; }
.footer-text { color: #aeaeb2 !important; }
.footer-link { color: #aeaeb2 !important; }

@media (prefers-color-scheme: dark) {
  body, .body-bg { background-color: #0c0c10 !important; }
  .card-bg { background-color: #161620 !important; }
  .card-border { border: 1px solid #2c2c34 !important; }
  .header-border { border-bottom: 1px solid #2c2c34 !important; }
  .footer-border { border-top: 1px solid #2c2c34 !important; }
  .h1-text { color: #f0f0f5 !important; }
  .h2-text { color: #f0f0f5 !important; }
  .sub-text { color: #8888a0 !important; }
  .body-text { color: #b4b4c8 !important; }
  .muted-text { color: #8888a0 !important; }
  .note-bg { background-color: #1e1e2a !important; }
  .note-border { border: 1px solid #2c2c34 !important; }
  .note-text { color: #b4b4c8 !important; }
  .step-bg { background-color: #3b1114 !important; }
  .step-text { color: #f87171 !important; }
  .link { color: #f87171 !important; }
  .attach-bg { background-color: #1e1e2a !important; }
  .attach-border { border: 1px solid #2c2c34 !important; }
  .attach-text { color: #8888a0 !important; }
  .attach-name { color: #b4b4c8 !important; }
  .footer-text { color: #606078 !important; }
  .footer-link { color: #606078 !important; }
}
</style>
</head>
<body class="body-bg" style="margin:0;padding:0;background-color:#f2f2f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="body-bg" style="background-color:#f2f2f7;">
<tr><td align="center" style="padding:32px 16px;">

<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="560"><tr><td><![endif]-->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="card-bg card-border" style="background-color:#ffffff;border:1px solid #e5e5ea;border-radius:12px;max-width:560px;">

<!-- Header -->
<tr><td class="header-border" style="padding:28px 32px 20px;border-bottom:1px solid #e5e5ea;">
  <span class="h1-text" style="font-size:18px;font-weight:700;color:#1c1c1e;letter-spacing:-0.3px;">WireGuard Configuration</span>
  {{if .PeerName}}<br><span class="sub-text" style="font-size:13px;color:#8e8e93;margin-top:4px;display:inline-block;">{{.PeerName}}</span>{{end}}
</td></tr>

{{if .Note}}
<!-- Custom Note -->
<tr><td style="padding:24px 32px 0;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="note-bg note-border" style="background-color:#f2f2f7;border:1px solid #e5e5ea;border-radius:8px;">
  <tr><td style="padding:16px 20px;">
    <span class="note-text" style="font-size:13px;color:#3a3a3c;white-space:pre-wrap;line-height:1.5;">{{.Note}}</span>
  </td></tr>
  </table>
</td></tr>
{{end}}

{{if .HasQR}}
<!-- QR Code -->
<tr><td style="padding:24px 32px 0;" align="center">
  <span class="muted-text" style="display:block;font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1.2px;margin-bottom:12px;">Scan to Connect</span>
  <img src="cid:qrcode" width="200" height="200" alt="QR Code" style="border-radius:8px;display:block;margin:0 auto;">
</td></tr>
{{end}}

<!-- Setup Instructions -->
<tr><td style="padding:24px 32px;">
  <span class="muted-text" style="display:block;font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1.2px;margin-bottom:16px;">Setup Instructions</span>

  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
  <tr><td style="padding-bottom:14px;">
    <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
      <td width="28" style="width:28px;vertical-align:top;padding-top:1px;">
        <!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" style="width:28px;height:28px;v-text-anchor:middle;" arcsize="50%" fillcolor="#fee2e2" stroke="f"><v:textbox inset="0,0,0,0"><center style="font-size:12px;font-weight:700;color:#dc2626;">1</center></v:textbox></v:roundrect><![endif]-->
        <!--[if !mso]><!--><div class="step-bg" style="width:28px;height:28px;background-color:#fee2e2;border-radius:14px;text-align:center;line-height:28px;font-size:12px;font-weight:700;mso-hide:all;"><span class="step-text" style="color:#dc2626;">1</span></div><!--<![endif]-->
      </td>
      <td style="padding-left:12px;">
        <span class="h2-text" style="font-size:13px;font-weight:600;color:#1c1c1e;">Install WireGuard</span><br>
        <span class="muted-text" style="font-size:12px;color:#8e8e93;">Download for
          <a href="https://itunes.apple.com/us/app/wireguard/id1451685025?ls=1&mt=12" class="link" style="color:#dc2626;text-decoration:none;">macOS</a> ·
          <a href="https://itunes.apple.com/us/app/wireguard/id1441195209?ls=1&mt=8" class="link" style="color:#dc2626;text-decoration:none;">iOS</a> ·
          <a href="https://play.google.com/store/apps/details?id=com.wireguard.android" class="link" style="color:#dc2626;text-decoration:none;">Android</a> ·
          <a href="https://download.wireguard.com/windows-client/wireguard-installer.exe" class="link" style="color:#dc2626;text-decoration:none;">Windows</a>
        </span>
      </td>
    </tr></table>
  </td></tr>
  <tr><td style="padding-bottom:14px;">
    <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
      <td width="28" style="width:28px;vertical-align:top;padding-top:1px;">
        <!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" style="width:28px;height:28px;v-text-anchor:middle;" arcsize="50%" fillcolor="#fee2e2" stroke="f"><v:textbox inset="0,0,0,0"><center style="font-size:12px;font-weight:700;color:#dc2626;">2</center></v:textbox></v:roundrect><![endif]-->
        <!--[if !mso]><!--><div class="step-bg" style="width:28px;height:28px;background-color:#fee2e2;border-radius:14px;text-align:center;line-height:28px;font-size:12px;font-weight:700;mso-hide:all;"><span class="step-text" style="color:#dc2626;">2</span></div><!--<![endif]-->
      </td>
      <td style="padding-left:12px;">
        <span class="h2-text" style="font-size:13px;font-weight:600;color:#1c1c1e;">Add Tunnel</span><br>
        <span class="muted-text" style="font-size:12px;color:#8e8e93;">Open WireGuard, tap + or "Add Tunnel", then scan the QR code above or import the attached .conf file.</span>
      </td>
    </tr></table>
  </td></tr>
  <tr><td>
    <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
      <td width="28" style="width:28px;vertical-align:top;padding-top:1px;">
        <!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" style="width:28px;height:28px;v-text-anchor:middle;" arcsize="50%" fillcolor="#fee2e2" stroke="f"><v:textbox inset="0,0,0,0"><center style="font-size:12px;font-weight:700;color:#dc2626;">3</center></v:textbox></v:roundrect><![endif]-->
        <!--[if !mso]><!--><div class="step-bg" style="width:28px;height:28px;background-color:#fee2e2;border-radius:14px;text-align:center;line-height:28px;font-size:12px;font-weight:700;mso-hide:all;"><span class="step-text" style="color:#dc2626;">3</span></div><!--<![endif]-->
      </td>
      <td style="padding-left:12px;">
        <span class="h2-text" style="font-size:13px;font-weight:600;color:#1c1c1e;">Activate</span><br>
        <span class="muted-text" style="font-size:12px;color:#8e8e93;">Toggle the tunnel on. A handshake within a few seconds confirms the connection is active.</span>
      </td>
    </tr></table>
  </td></tr>
  </table>
</td></tr>

<!-- Attachment note -->
<tr><td style="padding:0 32px 24px;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="attach-bg attach-border" style="background-color:#f2f2f7;border:1px solid #e5e5ea;border-radius:8px;">
  <tr><td style="padding:12px 16px;">
    <span class="attach-text" style="font-size:12px;color:#8e8e93;">📎 The configuration file <strong class="attach-name" style="color:#3a3a3c;">{{.Filename}}</strong> is attached to this email.</span>
  </td></tr>
  </table>
</td></tr>

<!-- Footer -->
<tr><td class="footer-border" style="padding:16px 32px;border-top:1px solid #e5e5ea;">
  <span class="footer-text" style="font-size:11px;color:#aeaeb2;">Sent from <a href="{{.ServerURL}}" class="footer-link" style="color:#aeaeb2;text-decoration:none;">wgRift</a></span>
</td></tr>

</table>
<!--[if mso]></td></tr></table><![endif]-->

</td></tr>
</table>
</body>
</html>`))

type templateData struct {
	PeerName  string
	Note      string
	HasQR     bool
	Filename  string
	ServerURL string
}

func renderPeerConfigEmail(p PeerConfigEmail) (string, error) {
	filename := "wgrift.conf"
	if p.PeerName != "" {
		filename = p.PeerName + ".conf"
	}

	serverURL := p.ServerName
	if serverURL == "" {
		serverURL = "#"
	}

	data := templateData{
		PeerName:  p.PeerName,
		Note:      p.Note,
		HasQR:     len(p.QRCodePNG) > 0,
		Filename:  filename,
		ServerURL: serverURL,
	}

	var buf bytes.Buffer
	if err := peerConfigTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering email template: %w", err)
	}
	return buf.String(), nil
}
