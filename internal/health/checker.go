package health

import (
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	defaultTimeout     = 3 * time.Second
	defaultConcurrency = 50
)

// Result holds the health check result for a single node.
type Result struct {
	ID      int64
	Name    string
	Server  string
	Port    string
	Alive   bool
	Latency time.Duration
}

// CheckNode performs a TCP (or UDP) connectivity test on a single proxy node.
func CheckNode(server, port string, proxyType string, udp bool) (bool, time.Duration) {
	address := net.JoinHostPort(server, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", address, defaultTimeout)
	if err == nil {
		conn.Close()
		return true, time.Since(start)
	}

	// For hysteria2 or explicitly UDP-marked nodes, try UDP
	if proxyType == "hysteria2" || udp {
		conn, err = net.DialTimeout("udp", address, defaultTimeout)
		if err == nil {
			conn.Close()
			return true, time.Since(start)
		}
	}

	return false, 0
}

// ProxyNode is the minimal proxy descriptor passed to BatchCheck.
type ProxyNode struct {
	ID     int64
	Name   string
	Server string
	Port   string
	Type   string
	UDP    bool
}

// BatchCheck concurrently checks multiple nodes and returns results sorted by latency (ascending).
func BatchCheck(nodes []ProxyNode) []Result {
	results := make([]Result, 0, len(nodes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, defaultConcurrency)

	for _, node := range nodes {
		wg.Add(1)
		go func(n ProxyNode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			alive, latency := CheckNode(n.Server, n.Port, n.Type, n.UDP)
			if alive {
				mu.Lock()
				results = append(results, Result{
					ID:      n.ID,
					Name:    n.Name,
					Server:  n.Server,
					Port:    n.Port,
					Alive:   true,
					Latency: latency,
				})
				mu.Unlock()
			} else {
				slog.Debug("node unreachable", "name", n.Name, "addr", fmt.Sprintf("%s:%s", n.Server, n.Port))
			}
		}(node)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
	})
	return results
}
