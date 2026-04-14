package confgen

import (
	"strings"
	"testing"
)

func TestGeneratePeerConf(t *testing.T) {
	conf := GeneratePeerConf(PeerConfParams{
		PrivateKey:          "cGVlci1wcml2YXRlLWtleQ==",
		Address:             "10.100.0.2/32",
		DNS:                 "1.1.1.1, 8.8.8.8",
		ServerPublicKey:     "c2VydmVyLXB1YmxpYy1rZXk=",
		ServerEndpoint:      "vpn.example.com:51820",
		AllowedIPs:          "0.0.0.0/0, ::/0",
		PersistentKeepalive: 25,
		MTU:                 1420,
	})

	checks := []string{
		"[Interface]",
		"PrivateKey = cGVlci1wcml2YXRlLWtleQ==",
		"Address = 10.100.0.2/32",
		"DNS = 1.1.1.1, 8.8.8.8",
		"MTU = 1420",
		"[Peer]",
		"PublicKey = c2VydmVyLXB1YmxpYy1rZXk=",
		"Endpoint = vpn.example.com:51820",
		"AllowedIPs = 0.0.0.0/0, ::/0",
		"PersistentKeepalive = 25",
	}

	for _, check := range checks {
		if !strings.Contains(conf, check) {
			t.Errorf("config missing %q\nGot:\n%s", check, conf)
		}
	}
}

func TestGeneratePeerConfWithPSK(t *testing.T) {
	conf := GeneratePeerConf(PeerConfParams{
		PrivateKey:      "a2V5",
		Address:         "10.0.0.2/32",
		ServerPublicKey: "cHVi",
		AllowedIPs:      "0.0.0.0/0",
		PresharedKey:    "cHNr",
	})

	if !strings.Contains(conf, "PresharedKey = cHNr") {
		t.Errorf("config missing PresharedKey\nGot:\n%s", conf)
	}
}

func TestGenerateServerConf(t *testing.T) {
	conf := GenerateServerConf(ServerConfParams{
		PrivateKey: "c2VydmVyLXByaXZhdGU=",
		Address:    "10.100.0.1/24",
		ListenPort: 51820,
		MTU:        1420,
		DNS:        "1.1.1.1",
		Peers: []ServerPeerBlock{
			{
				PublicKey:  "cGVlci0x",
				AllowedIPs: "10.100.0.2/32",
			},
			{
				PublicKey:           "cGVlci0y",
				AllowedIPs:          "10.100.0.3/32",
				PresharedKey:        "cHNr",
				PersistentKeepalive: 25,
			},
		},
	})

	checks := []string{
		"[Interface]",
		"ListenPort = 51820",
		"Address = 10.100.0.1/24",
		"[Peer]",
		"PublicKey = cGVlci0x",
		"PublicKey = cGVlci0y",
		"PresharedKey = cHNr",
		"PersistentKeepalive = 25",
	}

	for _, check := range checks {
		if !strings.Contains(conf, check) {
			t.Errorf("server config missing %q\nGot:\n%s", check, conf)
		}
	}
}

func TestGenerateServerConf_PostUpDown(t *testing.T) {
	conf := GenerateServerConf(ServerConfParams{
		PrivateKey: "c2VydmVyLXByaXZhdGU=",
		Address:    "10.100.0.1/24",
		ListenPort: 51820,
		PostUp:     "iptables -A FORWARD -i %i -j ACCEPT\niptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
		PostDown:   "iptables -D FORWARD -i %i -j ACCEPT",
	})

	checks := []string{
		"PostUp = iptables -A FORWARD -i %i -j ACCEPT",
		"PostUp = iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
		"PostDown = iptables -D FORWARD -i %i -j ACCEPT",
	}

	for _, check := range checks {
		if !strings.Contains(conf, check) {
			t.Errorf("server config missing %q\nGot:\n%s", check, conf)
		}
	}
}

func TestGenerateServerConf_NoPostUpDown(t *testing.T) {
	conf := GenerateServerConf(ServerConfParams{
		PrivateKey: "c2VydmVyLXByaXZhdGU=",
		Address:    "10.100.0.1/24",
		ListenPort: 51820,
	})

	if strings.Contains(conf, "PostUp") {
		t.Errorf("config should not contain PostUp when empty\nGot:\n%s", conf)
	}
	if strings.Contains(conf, "PostDown") {
		t.Errorf("config should not contain PostDown when empty\nGot:\n%s", conf)
	}
}

func TestGenerateStrippedConf_OmitsPostUpDown(t *testing.T) {
	conf := GenerateStrippedConf(ServerConfParams{
		PrivateKey: "c2VydmVyLXByaXZhdGU=",
		Address:    "10.100.0.1/24",
		ListenPort: 51820,
		PostUp:     "iptables -A FORWARD -i %i -j ACCEPT",
		PostDown:   "iptables -D FORWARD -i %i -j ACCEPT",
	})

	if strings.Contains(conf, "PostUp") {
		t.Errorf("stripped config should not contain PostUp\nGot:\n%s", conf)
	}
	if strings.Contains(conf, "PostDown") {
		t.Errorf("stripped config should not contain PostDown\nGot:\n%s", conf)
	}
}
