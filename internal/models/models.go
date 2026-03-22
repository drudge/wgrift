package models

import "time"

type InterfaceType string

const (
	InterfaceTypeSiteToSite    InterfaceType = "site-to-site"
	InterfaceTypeClientAccess  InterfaceType = "client-access"
)

type Interface struct {
	ID                 string        `json:"id"`
	Type               InterfaceType `json:"type"`
	ListenPort         int           `json:"listen_port"`
	PrivateKeyEncrypted string       `json:"private_key_encrypted"`
	Address            string        `json:"address"`
	DNS                string        `json:"dns"`
	MTU                int           `json:"mtu"`
	Enabled            bool          `json:"enabled"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type Peer struct {
	ID                    string    `json:"id"`
	InterfaceID           string    `json:"interface_id"`
	Name                  string    `json:"name"`
	PublicKey             string    `json:"public_key"`
	PrivateKeyEncrypted   string    `json:"private_key_encrypted"`
	PresharedKeyEncrypted string    `json:"preshared_key_encrypted,omitempty"`
	AllowedIPs            string    `json:"allowed_ips"`
	Endpoint              string    `json:"endpoint,omitempty"`
	PersistentKeepalive   int       `json:"persistent_keepalive"`
	Enabled               bool      `json:"enabled"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	LastHandshake         *time.Time `json:"last_handshake,omitempty"`
	TransferRx            int64     `json:"transfer_rx"`
	TransferTx            int64     `json:"transfer_tx"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
