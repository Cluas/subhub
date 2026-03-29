package proxy

// Node represents a proxy server that can establish connections.
// It is protocol-agnostic -- whether vless or trojan, it is simply a usable node.
type Node interface {
	ID() int64
	Name() string
	// RawConfig returns the full configuration of the node.
	// Fields are written by the Parser and read by the Renderer; the core system does not parse them.
	RawConfig() map[string]any
}
