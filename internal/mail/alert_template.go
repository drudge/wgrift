package mail

import (
	"bytes"
	"fmt"
	"html/template"
)

var alertTmpl = template.Must(template.New("alert").Parse(`<!DOCTYPE html>
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
body, .body-bg { background-color: transparent !important; }
.card-bg { background-color: #ffffff !important; }
.card-border { border: 1px solid #e5e5ea !important; }
.header-border { border-bottom: 1px solid #e5e5ea !important; }
.footer-border { border-top: 1px solid #e5e5ea !important; }
.h1-text { color: #1c1c1e !important; }
.sub-text { color: #8e8e93 !important; }
.body-text { color: #3a3a3c !important; }
.muted-text { color: #8e8e93 !important; }
.detail-bg { background-color: #f2f2f7 !important; }
.detail-border { border: 1px solid #e5e5ea !important; }
.detail-label { color: #8e8e93 !important; }
.detail-value { color: #3a3a3c !important; }
.badge-connect-bg { background-color: #dcfce7 !important; }
.badge-connect-text { color: #166534 !important; }
.badge-disconnect-bg { background-color: #fee2e2 !important; }
.badge-disconnect-text { color: #991b1b !important; }
.footer-text { color: #aeaeb2 !important; }
.footer-link { color: #aeaeb2 !important; }

@media (prefers-color-scheme: dark) {
  body, .body-bg { background-color: transparent !important; }
  .card-bg { background-color: #161620 !important; }
  .card-border { border: 1px solid #2c2c34 !important; }
  .header-border { border-bottom: 1px solid #2c2c34 !important; }
  .footer-border { border-top: 1px solid #2c2c34 !important; }
  .h1-text { color: #f0f0f5 !important; }
  .sub-text { color: #8888a0 !important; }
  .body-text { color: #b4b4c8 !important; }
  .muted-text { color: #8888a0 !important; }
  .detail-bg { background-color: #1e1e2a !important; }
  .detail-border { border: 1px solid #2c2c34 !important; }
  .detail-label { color: #8888a0 !important; }
  .detail-value { color: #b4b4c8 !important; }
  .badge-connect-bg { background-color: #14532d !important; }
  .badge-connect-text { color: #86efac !important; }
  .badge-disconnect-bg { background-color: #3b1114 !important; }
  .badge-disconnect-text { color: #fca5a5 !important; }
  .footer-text { color: #606078 !important; }
  .footer-link { color: #606078 !important; }
}
</style>
</head>
<body class="body-bg" style="margin:0;padding:0;background-color:transparent;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="body-bg" style="background-color:transparent;">
<tr><td align="center" style="padding:32px 16px;">

<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="560"><tr><td><![endif]-->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="card-bg card-border" style="background-color:#ffffff;border:1px solid #e5e5ea;border-radius:12px;max-width:560px;">

<!-- Header -->
<tr><td class="header-border" style="padding:28px 32px 20px;border-bottom:1px solid #e5e5ea;">
  <span class="h1-text" style="font-size:18px;font-weight:700;color:#1c1c1e;letter-spacing:-0.3px;">{{.PeerName}} {{if .IsConnect}}Connected{{else}}Disconnected{{end}}</span>
  <br><span class="sub-text" style="font-size:12px;color:#8e8e93;margin-top:4px;display:inline-block;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,monospace;">{{.PublicKey}}</span>
</td></tr>

<!-- Status Message -->
<tr><td style="padding:24px 32px 0;">
  {{if .IsConnect}}
  <span class="badge-connect-bg badge-connect-text" style="display:inline-block;background-color:#dcfce7;color:#166534;font-size:13px;font-weight:500;padding:8px 16px;border-radius:8px;line-height:1.4;">{{.Message}}</span>
  {{else}}
  <span class="badge-disconnect-bg badge-disconnect-text" style="display:inline-block;background-color:#fee2e2;color:#991b1b;font-size:13px;font-weight:500;padding:8px 16px;border-radius:8px;line-height:1.4;">{{.Message}}</span>
  {{end}}
</td></tr>

<!-- Details -->
<tr><td style="padding:20px 32px 24px;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="detail-bg detail-border" style="background-color:#f2f2f7;border:1px solid #e5e5ea;border-radius:8px;">
  {{if .Endpoint}}
  <tr><td style="padding:12px 16px 0;">
    <span class="detail-label" style="font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1px;">Endpoint</span><br>
    <span class="detail-value" style="font-size:13px;color:#3a3a3c;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,monospace;">{{.Endpoint}}</span>
  </td></tr>
  {{end}}
  {{if .HasTransfer}}
  <tr><td style="padding:12px 16px 0;">
    <span class="detail-label" style="font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1px;">Transfer</span><br>
    <span class="detail-value" style="font-size:13px;color:#3a3a3c;">{{.TransferRx}} received, {{.TransferTx}} sent</span>
  </td></tr>
  {{end}}
  <tr><td style="padding:12px 16px;">
    <span class="detail-label" style="font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1px;">Time</span><br>
    <span class="detail-value" style="font-size:13px;color:#3a3a3c;">{{.Timestamp}}</span>
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

type alertTemplateData struct {
	PeerName      string
	PublicKey     string
	InterfaceName string
	IsConnect     bool
	Message       string
	Endpoint      string
	TransferRx    string
	TransferTx    string
	HasTransfer   bool
	Timestamp     string
	ServerURL     string
}

func renderAlertEmail(a AlertEmail) (string, error) {
	serverURL := a.ServerName
	if serverURL == "" {
		serverURL = "#"
	}

	var message string
	if a.Event == "connected" {
		message = a.PeerName + " has connected to the tunnel and is now online."
	} else {
		message = a.PeerName + " has disconnected from the tunnel and is no longer reachable."
	}

	data := alertTemplateData{
		PeerName:      a.PeerName,
		PublicKey:     a.PublicKey,
		InterfaceName: a.InterfaceName,
		IsConnect:     a.Event == "connected",
		Message:       message,
		Endpoint:      a.Endpoint,
		TransferRx:    formatBytes(a.TransferRx),
		TransferTx:    formatBytes(a.TransferTx),
		HasTransfer:   a.TransferRx > 0 || a.TransferTx > 0,
		Timestamp:     a.Timestamp,
		ServerURL:     serverURL,
	}

	var buf bytes.Buffer
	if err := alertTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
