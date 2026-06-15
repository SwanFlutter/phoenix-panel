package links

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/phoenix-panel/phoenix/internal/models"
)

func testUser() models.User {
	return models.User{
		Username:       "alice",
		UUID:           "11111111-2222-3333-4444-555555555555",
		TrojanPassword: "trojanpw",
		SSPassword:     "sspw",
		SSMethod:       "chacha20-ietf-poly1305",
	}
}

func testNode() models.Node {
	return models.Node{Address: "example.com", Core: models.CoreXray, IsActive: true}
}

func TestBuildVLESS(t *testing.T) {
	in := models.Inbound{Protocol: models.ProtoVLESS, Port: 443, Network: "ws",
		Security: "tls", SNI: "example.com", Path: "/ws", IsActive: true}
	uri, err := Build(testUser(), in, testNode(), "alice-vless")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasPrefix(uri, "vless://") {
		t.Fatalf("expected vless:// prefix, got %q", uri)
	}
	for _, want := range []string{"example.com:443", "type=ws", "security=tls", "sni=example.com"} {
		if !strings.Contains(uri, want) {
			t.Errorf("vless uri missing %q: %s", want, uri)
		}
	}
}

func TestBuildVMessIsBase64JSON(t *testing.T) {
	in := models.Inbound{Protocol: models.ProtoVMess, Port: 80, Network: "tcp", IsActive: true}
	uri, err := Build(testUser(), in, testNode(), "alice-vmess")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	payload := strings.TrimPrefix(uri, "vmess://")
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		t.Fatalf("vmess payload is not valid base64: %v", err)
	}
}

func TestSupportedByFiltersHysteria2OnXray(t *testing.T) {
	// Hysteria2 is sing-box only; BuildAll must skip it on an Xray node.
	node := testNode() // xray
	pairs := []NodeInbound{
		{Node: node, Inbound: models.Inbound{Protocol: models.ProtoHysteria2, Port: 443, IsActive: true}},
		{Node: node, Inbound: models.Inbound{Protocol: models.ProtoVLESS, Port: 443, IsActive: true}},
	}
	uris := BuildAll(testUser(), pairs)
	if len(uris) != 1 {
		t.Fatalf("expected 1 link (vless only), got %d: %v", len(uris), uris)
	}
	if !strings.HasPrefix(uris[0], "vless://") {
		t.Fatalf("expected the surviving link to be vless, got %q", uris[0])
	}
}

func TestRenderBase64RoundTrips(t *testing.T) {
	uris := []string{"vless://a", "trojan://b"}
	out := Render(uris, FormatBase64)
	dec, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(dec) != "vless://a\ntrojan://b" {
		t.Fatalf("unexpected decoded body: %q", dec)
	}
}
