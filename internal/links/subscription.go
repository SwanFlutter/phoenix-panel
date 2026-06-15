package links

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// SubFormat enumerates supported subscription output encodings.
type SubFormat string

const (
	// FormatBase64 is the universal base64-of-newline-joined-URIs format
	// understood by v2rayN, v2rayNG, Nekoray, etc.
	FormatBase64 SubFormat = "base64"
	// FormatPlain returns the raw newline-joined URIs (some clients prefer this).
	FormatPlain SubFormat = "plain"
)

// NodeInbound pairs an inbound with the node that hosts it, as needed to build
// a connectable link.
type NodeInbound struct {
	Node    models.Node
	Inbound models.Inbound
}

// BuildAll returns every share URL for a user across the provided node/inbound
// pairs, skipping any the protocol can't build. The label is derived from the
// inbound tag and node name so users can tell links apart in their client.
func BuildAll(user models.User, pairs []NodeInbound) []string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if !p.Inbound.IsActive || !p.Node.IsActive {
			continue
		}
		// Only emit links the inbound's core can actually serve.
		if !p.Inbound.Protocol.SupportedBy(p.Node.Core) {
			continue
		}
		label := fmt.Sprintf("%s-%s", user.Username, p.Inbound.Tag)
		uri, err := Build(user, p.Inbound, p.Node, label)
		if err != nil {
			continue
		}
		out = append(out, uri)
	}
	return out
}

// Render encodes the URIs into the requested subscription format.
func Render(uris []string, format SubFormat) string {
	joined := strings.Join(uris, "\n")
	switch format {
	case FormatPlain:
		return joined
	case FormatBase64:
		fallthrough
	default:
		return base64.StdEncoding.EncodeToString([]byte(joined))
	}
}
