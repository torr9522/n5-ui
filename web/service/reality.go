package service

import (
	"crypto/rand"
	"encoding/hex"
	"x-ui/util/common"
	"x-ui/xray"
)

type RealityService struct {
}

type RealityDefaultConfig struct {
	Dest        string   `json:"dest"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"publicKey"`
	SpiderX     string   `json:"spiderX"`
}

func (s *RealityService) GenerateX25519KeyPair() (*xray.X25519KeyPair, error) {
	return xray.GenerateX25519KeyPair()
}

func (s *RealityService) GenerateDefaultConfig() (*RealityDefaultConfig, error) {
	keyPair, err := xray.GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	shortId, err := randomShortId()
	if err != nil {
		return nil, err
	}

	return &RealityDefaultConfig{
		Dest:        "www.cloudflare.com:443",
		ServerNames: []string{"www.cloudflare.com"},
		PrivateKey:  keyPair.PrivateKey,
		ShortIds:    []string{shortId},
		Fingerprint: "chrome",
		PublicKey:   keyPair.PublicKey,
		SpiderX:     "/",
	}, nil
}

func randomShortId() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", common.NewErrorf("shortId generate failed")
	}
	return hex.EncodeToString(b), nil
}
