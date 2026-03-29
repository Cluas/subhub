package subscription

import "github.com/Cluas/subhub/internal/node"

// nodeResult implements proxy.Node for a single Mihomo proxy entry.
type nodeResult struct {
	id     int64
	name   string
	config map[string]any
}

var _ proxy.Node = (*nodeResult)(nil)

func (n *nodeResult) ID() int64                { return n.id }
func (n *nodeResult) Name() string             { return n.name }
func (n *nodeResult) RawConfig() map[string]any { return n.config }
