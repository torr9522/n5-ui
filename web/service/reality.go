package service

import "x-ui/xray"

type RealityService struct {
}

func (s *RealityService) GenerateX25519KeyPair() (*xray.X25519KeyPair, error) {
	return xray.GenerateX25519KeyPair()
}
