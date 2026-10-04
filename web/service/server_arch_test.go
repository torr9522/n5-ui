package service

import "testing"

func TestXrayReleaseAssetName(t *testing.T) {
	tests := []struct {
		arch string
		want string
		err  bool
	}{
		{arch: "amd64", want: "Xray-linux-64.zip"},
		{arch: "arm64", want: "Xray-linux-arm64.zip"},
		{arch: "386", err: true},
	}

	for _, test := range tests {
		t.Run(test.arch, func(t *testing.T) {
			got, err := xrayReleaseAssetName(test.arch)
			if (err != nil) != test.err {
				t.Fatalf("xrayReleaseAssetName(%q) error = %v", test.arch, err)
			}
			if got != test.want {
				t.Fatalf("xrayReleaseAssetName(%q) = %q, want %q", test.arch, got, test.want)
			}
		})
	}
}
