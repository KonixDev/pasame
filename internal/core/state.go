package core

import (
	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/session"
)

type FileView struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

type State struct {
	Phase           string           `json:"phase"`
	Name            string           `json:"name"`
	Files           []FileView       `json:"files"`
	Count           int              `json:"count"`
	Total           string           `json:"total"`
	ReceiveOnly     bool             `json:"receiveOnly"`
	Unreadable      []string         `json:"unreadable"`
	Addresses       []addr.Address   `json:"addresses"`
	QR              string           `json:"qr"`
	SharedAt        int64            `json:"sharedAt"`
	Stats           session.Snapshot `json:"stats"`
	VPN             bool             `json:"vpn"`
	Ifaces          []addr.Candidate `json:"ifaces"`
	NetChanged      bool             `json:"netChanged"`
	FirewallHint    string           `json:"firewallHint"`
	PickUnsupported bool             `json:"pickUnsupported"`
	Strict          bool             `json:"strict"`
	Quarantine      string           `json:"quarantine"`
	Version         string           `json:"version"`
}
