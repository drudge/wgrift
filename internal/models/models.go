package models

import "time"

type InterfaceType string

const (
	InterfaceTypeSiteToSite   InterfaceType = "site-to-site"
	InterfaceTypeClientAccess InterfaceType = "client-access"
)

type PeerType string

const (
	PeerTypeClient PeerType = "client"
	PeerTypeSite   PeerType = "site"
)

type Interface struct {
	ID                  string        `json:"id"`
	Type                InterfaceType `json:"type"`
	ListenPort          int           `json:"listen_port"`
	PrivateKeyEncrypted string        `json:"private_key_encrypted"`
	Address             string        `json:"address"`
	DNS                 string        `json:"dns"`
	MTU                 int           `json:"mtu"`
	Endpoint            string        `json:"endpoint"`
	Enabled             bool          `json:"enabled"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

type Peer struct {
	ID                    string     `json:"id"`
	InterfaceID           string     `json:"interface_id"`
	Type                  PeerType   `json:"type"`
	Name                  string     `json:"name"`
	PublicKey             string     `json:"public_key"`
	PrivateKeyEncrypted   string     `json:"private_key_encrypted"`
	PresharedKeyEncrypted string     `json:"preshared_key_encrypted,omitempty"`
	Address               string     `json:"address"`
	AllowedIPs            string     `json:"allowed_ips"`
	ClientAllowedIPs      string     `json:"client_allowed_ips"`
	DNS                   string     `json:"dns"`
	Endpoint              string     `json:"endpoint,omitempty"`
	PersistentKeepalive   int        `json:"persistent_keepalive"`
	Enabled               bool       `json:"enabled"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	LastHandshake         *time.Time `json:"last_handshake,omitempty"`
	TransferRx            int64      `json:"transfer_rx"`
	TransferTx            int64      `json:"transfer_tx"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	IsInitial    bool      `json:"is_initial"`
	OIDCProvider string    `json:"oidc_provider,omitempty"`
	OIDCSubject  string    `json:"oidc_subject,omitempty"`
	OIDCIssuer   string    `json:"oidc_issuer,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OIDCProvider struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Issuer                string    `json:"issuer"`
	ClientID              string    `json:"client_id"`
	ClientSecretEncrypted string    `json:"-"`
	Scopes                string    `json:"scopes"`
	AutoDiscover          bool      `json:"auto_discover"`
	AdminClaim            string    `json:"admin_claim"`
	AdminValue            string    `json:"admin_value"`
	DefaultRole           string    `json:"default_role"`
	Enabled               bool      `json:"enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type OIDCState struct {
	State      string    `json:"state"`
	ProviderID string    `json:"provider_id"`
	Nonce      string    `json:"nonce"`
	CreatedAt  time.Time `json:"created_at"`
}

type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CSRFToken  string    `json:"-"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
}

type ConnectionLog struct {
	ID          int64     `json:"id"`
	PeerID      string    `json:"peer_id"`
	PeerName    string    `json:"peer_name"`
	InterfaceID string    `json:"interface_id"`
	Event       string    `json:"event"`
	TransferRx  int64     `json:"transfer_rx"`
	TransferTx  int64     `json:"transfer_tx"`
	RecordedAt  time.Time `json:"recorded_at"`
}
