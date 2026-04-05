package mail

import (
	"bytes"
	"html/template"
	"time"
)

var testTmpl = template.Must(template.New("test").Parse(`<!DOCTYPE html>
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
.card-bg { background-color: transparent !important; }
.card-border { border: 1px solid #e5e5ea !important; }
.header-border { border-bottom: 1px solid #e5e5ea !important; }
.footer-border { border-top: 1px solid #e5e5ea !important; }
.h1-text { color: #1c1c1e !important; }
.sub-text { color: #8e8e93 !important; }
.body-text { color: #3a3a3c !important; }
.muted-text { color: #8e8e93 !important; }
.detail-bg { background-color: transparent !important; }
.detail-border { border: 1px solid #e5e5ea !important; }
.detail-label { color: #8e8e93 !important; }
.detail-value { color: #3a3a3c !important; }
.check-circle { background-color: #dcfce7 !important; }
.check-icon { color: #16a34a !important; }
.success-text { color: #166534 !important; }
.footer-text { color: #aeaeb2 !important; }
.footer-link { color: #aeaeb2 !important; }

@media (prefers-color-scheme: dark) {
  body, .body-bg { background-color: transparent !important; }
  .card-bg { background-color: transparent !important; }
  .card-border { border: 1px solid #2c2c34 !important; }
  .header-border { border-bottom: 1px solid #2c2c34 !important; }
  .footer-border { border-top: 1px solid #2c2c34 !important; }
  .h1-text { color: #f0f0f5 !important; }
  .sub-text { color: #8888a0 !important; }
  .body-text { color: #b4b4c8 !important; }
  .muted-text { color: #8888a0 !important; }
  .detail-bg { background-color: transparent !important; }
  .detail-border { border: 1px solid #2c2c34 !important; }
  .detail-label { color: #8888a0 !important; }
  .detail-value { color: #b4b4c8 !important; }
  .check-circle { background-color: #14532d !important; }
  .check-icon { color: #4ade80 !important; }
  .success-text { color: #86efac !important; }
  .footer-text { color: #606078 !important; }
  .footer-link { color: #606078 !important; }
}
</style>
</head>
<body class="body-bg" style="margin:0;padding:0;background-color:transparent;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="body-bg" style="background-color:transparent;">
<tr><td align="center" style="padding:32px 16px;">

<!--[if mso]><table role="presentation" cellpadding="0" cellspacing="0" border="0" width="560"><tr><td><![endif]-->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="card-bg card-border" style="background-color:transparent;border:1px solid #e5e5ea;border-radius:12px;max-width:560px;">

<!-- Header -->
<tr><td class="header-border" style="padding:28px 32px 20px;border-bottom:1px solid #e5e5ea;">
  <span class="h1-text" style="font-size:18px;font-weight:700;color:#1c1c1e;letter-spacing:-0.3px;">SMTP Test</span>
</td></tr>

<!-- Checkmark -->
<tr><td align="center" style="padding:32px 32px 0;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0">
  <tr><td align="center" class="check-circle" style="width:80px;height:80px;background-color:#dcfce7;border-radius:50%;text-align:center;vertical-align:middle;">
    <!--[if mso]>
    <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" style="width:80px;height:80px;v-text-anchor:middle;" arcsize="50%" fillcolor="#dcfce7" stroke="f">
    <v:textbox inset="0,0,0,0"><center>
    <![endif]-->
    <span class="check-icon" style="font-size:40px;line-height:80px;color:#16a34a;">&#10003;</span>
    <!--[if mso]>
    </center></v:textbox>
    </v:roundrect>
    <![endif]-->
  </td></tr>
  </table>
</td></tr>

<!-- Success Message -->
<tr><td align="center" style="padding:20px 32px 0;">
  <span class="success-text" style="font-size:15px;font-weight:600;color:#166534;">Configuration Verified</span>
  <br><span class="body-text" style="font-size:13px;color:#3a3a3c;margin-top:4px;display:inline-block;">Your SMTP configuration is working correctly. Emails from wgRift will be delivered to this address.</span>
</td></tr>

<!-- Details -->
<tr><td style="padding:20px 32px 24px;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="detail-bg detail-border" style="background-color:transparent;border:1px solid #e5e5ea;border-radius:8px;">
  <tr><td style="padding:12px 16px 0;">
    <span class="detail-label" style="font-size:11px;font-weight:600;color:#8e8e93;text-transform:uppercase;letter-spacing:1px;">Delivered To</span><br>
    <span class="detail-value" style="font-size:13px;color:#3a3a3c;">{{.To}}</span>
  </td></tr>
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

type testTemplateData struct {
	To        string
	Timestamp string
	ServerURL string
}

func renderTestEmail(serverURL, to string) (string, error) {
	if serverURL == "" {
		serverURL = "#"
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}

	data := testTemplateData{
		To:        to,
		Timestamp: time.Now().In(loc).Format("Jan 2, 2006 3:04:05 PM MST"),
		ServerURL: serverURL,
	}

	var buf bytes.Buffer
	if err := testTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
