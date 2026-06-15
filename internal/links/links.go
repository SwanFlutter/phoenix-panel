// Package links builds shareable proxy URLs (vless://, vmess://, trojan://, ss://,
// hysteria2://, tuic://) from a user + inbound pair, plus aggregated
// subscription documents in several client formats.
package links

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// Build returns a single share URL for the given user on the given inbound.
// The Node supplies the connect address; the Inbound supplies transport/security.
func Build(user models.User, inbound models.Inbound, node models.Node, label string) (string, error) {
	addr := node.Address
	if addr == "" {
		addr = inbound.Listen
	}
	switch inbound.Protocol {
	case models.ProtoVLESS:
		return buildVLESS(user, inbound, addr, label), nil
	case models.ProtoVMess:
		return buildVMess(user, inbound, addr, label), nil
	case models.ProtoTrojan:
		return buildTrojan(user, inbound, addr, label), nil
	case models.ProtoShadowsocks:
		return buildShadowsocks(user, inbound, addr, label), nil
	case models.ProtoHysteria2:
		return buildHysteria2(user, inbound, addr, label), nil
	case models.ProtoTUIC:
		return buildTUIC(user, inbound, addr, label), nil
	default:
		return "", fmt.Errorf("links: unsupported protocol %q", inbound.Protocol)
	}
}

// commonStreamParams appends the transport/TLS query params shared by the
// URI-style protocols (VLESS/Trojan).
func commonStreamParams(q url.Values, in models.Inbound) {
	q.Set("type", orDefault(in.Network, "tcp"))
	if in.Security != "" && in.Security != "none" {
		q.Set("security", in.Security)
	}
	if in.SNI != "" {
		q.Set("sni", in.SNI)
	}
	if in.Fingerprint != "" {
		q.Set("fp", in.Fingerprint)
	}
	if in.Flow != "" {
		q.Set("flow", in.Flow)
	}
	switch in.Network {
	case "ws":
		if in.Path != "" {
			q.Set("path", in.Path)
		}
		if in.Host != "" {
			q.Set("host", in.Host)
		}
	case "grpc":
		if in.Path != "" {
			q.Set("serviceName", in.Path)
		}
	case "http", "h2":
		if in.Path != "" {
			q.Set("path", in.Path)
		}
		if in.Host != "" {
			q.Set("host", in.Host)
		}
	}
	// REALITY public key / short id, if present.
	if in.Security == "reality" && in.RealitySettings != nil {
		if pbk, ok := in.RealitySettings["public_key"].(string); ok && pbk != "" {
			q.Set("pbk", pbk)
		}
		if sid, ok := in.RealitySettings["short_id"].(string); ok && sid != "" {
			q.Set("sid", sid)
		}
	}
}

func buildVLESS(u models.User, in models.Inbound, addr, label string) string {
	q := url.Values{}
	q.Set("encryption", "none")
	commonStreamParams(q, in)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		u.UUID, addr, in.Port, q.Encode(), url.PathEscape(label))
}

func buildTrojan(u models.User, in models.Inbound, addr, label string) string {
	q := url.Values{}
	commonStreamParams(q, in)
	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		url.QueryEscape(u.TrojanPassword), addr, in.Port, q.Encode(), url.PathEscape(label))
}

// buildVMess encodes the legacy base64-JSON vmess:// format (v2 schema).
func buildVMess(u models.User, in models.Inbound, addr, label string) string {
	cfg := map[string]any{
		"v":    "2",
		"ps":   label,
		"add":  addr,
		"port": strconv.Itoa(in.Port),
		"id":   u.UUID,
		"aid":  "0",
		"scy":  "auto",
		"net":  orDefault(in.Network, "tcp"),
		"type": "none",
		"host": in.Host,
		"path": in.Path,
		"tls":  tlsField(in.Security),
		"sni":  in.SNI,
		"fp":   in.Fingerprint,
	}
	raw, _ := json.Marshal(cfg)
	return "vmess://" + base64.StdEncoding.EncodeToString(raw)
}

// buildShadowsocks encodes the SIP002 ss:// URI.
func buildShadowsocks(u models.User, in models.Inbound, addr, label string) string {
	userinfo := base64.RawURLEncoding.EncodeToString(
		[]byte(orDefault(u.SSMethod, "chacha20-ietf-poly1305") + ":" + u.SSPassword))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userinfo, addr, in.Port, url.PathEscape(label))
}

func buildHysteria2(u models.User, in models.Inbound, addr, label string) string {
	q := url.Values{}
	if in.SNI != "" {
		q.Set("sni", in.SNI)
	}
	if in.Host != "" {
		q.Set("obfs", "") // placeholder for obfs config carried in Extra
	}
	enc := ""
	if len(q) > 0 {
		enc = "?" + q.Encode()
	}
	return fmt.Sprintf("hysteria2://%s@%s:%d%s#%s",
		url.QueryEscape(u.TrojanPassword), addr, in.Port, enc, url.PathEscape(label))
}

func buildTUIC(u models.User, in models.Inbound, addr, label string) string {
	q := url.Values{}
	if in.SNI != "" {
		q.Set("sni", in.SNI)
	}
	q.Set("congestion_control", "bbr")
	q.Set("alpn", "h3")
	return fmt.Sprintf("tuic://%s:%s@%s:%d?%s#%s",
		u.UUID, url.QueryEscape(u.TrojanPassword), addr, in.Port, q.Encode(), url.PathEscape(label))
}

func tlsField(security string) string {
	if security == "tls" || security == "reality" {
		return "tls"
	}
	return ""
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
