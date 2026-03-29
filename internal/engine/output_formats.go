package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/Cluas/subhub/internal/model"
)

// FormatProxyOutput renders proxies in the given format.
// Returns (body, contentType, error).
// Supported formats: "clash", "surge", "shadowrocket", "quantumultx", "loon", "singbox".
func FormatProxyOutput(format string, proxies []map[string]interface{}) ([]byte, string, error) {
	switch strings.ToLower(format) {
	case "clash":
		provider := model.Provider{Proxies: proxies}
		if provider.Proxies == nil {
			provider.Proxies = []map[string]interface{}{}
		}
		out, err := yaml.Marshal(provider)
		if err != nil {
			return nil, "", fmt.Errorf("clash marshal: %w", err)
		}
		return out, "text/yaml; charset=utf-8", nil

	case "shadowrocket":
		link := ProxiesToLinks(proxies)
		return []byte(link), "text/plain; charset=utf-8", nil

	case "surge":
		out := formatSurgeProxies(proxies)
		return []byte(out), "text/plain; charset=utf-8", nil

	case "quantumultx":
		var sb strings.Builder
		sb.WriteString("[Remote Proxy]\n")
		for _, p := range proxies {
			sb.WriteString(quantumultxProxyLine(p))
			sb.WriteString("\n")
		}
		return []byte(sb.String()), "text/plain; charset=utf-8", nil

	case "loon":
		var sb strings.Builder
		sb.WriteString("[Remote Proxy]\n")
		for _, p := range proxies {
			sb.WriteString(loonProxyLine(p))
			sb.WriteString("\n")
		}
		return []byte(sb.String()), "text/plain; charset=utf-8", nil

	case "singbox":
		outbounds := make([]map[string]interface{}, 0, len(proxies))
		for _, p := range proxies {
			if ob := singboxOutbound(p); ob != nil {
				outbounds = append(outbounds, ob)
			}
		}
		result := map[string]interface{}{"outbounds": outbounds}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("singbox marshal: %w", err)
		}
		return out, "application/json; charset=utf-8", nil

	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatRuleOutput renders rules in the given format.
// Returns (body, contentType, error).
// Supported formats: "clash", "surge", "shadowrocket", "quantumultx", "loon", "singbox".
func FormatRuleOutput(format string, rules []*model.Rule) ([]byte, string, error) {
	switch strings.ToLower(format) {
	case "clash", "surge", "shadowrocket":
		payload := make([]string, 0, len(rules))
		for _, r := range rules {
			if strings.EqualFold(r.Type, "MATCH") || r.Payload == "" {
				continue
			}
			payload = append(payload, r.Type+","+r.Payload)
		}
		provider := model.Provider{Payload: payload}
		ct := "text/yaml; charset=utf-8"
		if strings.ToLower(format) == "surge" {
			ct = "text/plain; charset=utf-8"
		}
		out, err := yaml.Marshal(provider)
		if err != nil {
			return nil, "", fmt.Errorf("rule marshal: %w", err)
		}
		return out, ct, nil

	case "quantumultx":
		// Plain text, one rule per line, no header.
		var sb strings.Builder
		for _, r := range rules {
			if strings.EqualFold(r.Type, "MATCH") || r.Payload == "" {
				continue
			}
			sb.WriteString(r.Type + "," + r.Payload + "\n")
		}
		return []byte(sb.String()), "text/plain; charset=utf-8", nil

	case "loon":
		// [Remote Filter] header + "TYPE, payload" lines.
		var sb strings.Builder
		sb.WriteString("[Remote Filter]\n")
		for _, r := range rules {
			if strings.EqualFold(r.Type, "MATCH") || r.Payload == "" {
				continue
			}
			sb.WriteString(r.Type + ", " + r.Payload + "\n")
		}
		return []byte(sb.String()), "text/plain; charset=utf-8", nil

	case "singbox":
		// {"version":2,"rules":[{"domain_suffix":[...],"domain_keyword":[...],"ip_cidr":[...]}]}
		domainSuffix := []string{}
		domainKeyword := []string{}
		ipCIDR := []string{}
		for _, r := range rules {
			if r.Payload == "" {
				continue
			}
			switch strings.ToUpper(r.Type) {
			case "DOMAIN-SUFFIX":
				domainSuffix = append(domainSuffix, r.Payload)
			case "DOMAIN-KEYWORD":
				domainKeyword = append(domainKeyword, r.Payload)
			case "IP-CIDR":
				ipCIDR = append(ipCIDR, r.Payload)
			// MATCH, GEOIP, RULE-SET skipped
			}
		}
		ruleEntry := map[string]interface{}{}
		if len(domainSuffix) > 0 {
			ruleEntry["domain_suffix"] = domainSuffix
		}
		if len(domainKeyword) > 0 {
			ruleEntry["domain_keyword"] = domainKeyword
		}
		if len(ipCIDR) > 0 {
			ruleEntry["ip_cidr"] = ipCIDR
		}
		result := map[string]interface{}{
			"version": 2,
			"rules":   []interface{}{ruleEntry},
		}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("singbox rule marshal: %w", err)
		}
		return out, "application/json; charset=utf-8", nil

	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// formatSurgeProxies converts proxy maps to Surge proxy list format.
func formatSurgeProxies(proxies []map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("[Proxy]\n")
	for _, p := range proxies {
		line := surgeProxyLine(p)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// surgeProxyLine returns a single Surge proxy line for the given proxy map.
func surgeProxyLine(p map[string]interface{}) string {
	name, _ := p["name"].(string)
	server, _ := p["server"].(string)
	port := portToString(p["port"])
	proxyType := strings.ToLower(fmt.Sprintf("%v", p["type"]))

	switch proxyType {
	case "ss":
		cipher, _ := p["cipher"].(string)
		password, _ := p["password"].(string)
		return fmt.Sprintf("%s = ss, %s, %s, encrypt-method=%s, password=%s",
			name, server, port, cipher, password)

	case "vmess":
		uuid, _ := p["uuid"].(string)
		line := fmt.Sprintf("%s = vmess, %s, %s, username=%s", name, server, port, uuid)
		if tls, _ := p["tls"].(bool); tls {
			line += ", tls=true"
		}
		if net, _ := p["network"].(string); net == "ws" {
			line += ", ws=true"
			if wsOpts, ok := p["ws-opts"].(map[string]interface{}); ok {
				if wsPath, _ := wsOpts["path"].(string); wsPath != "" {
					line += ", ws-path=" + wsPath
				}
			}
		}
		return line

	case "trojan":
		password, _ := p["password"].(string)
		line := fmt.Sprintf("%s = trojan, %s, %s, password=%s", name, server, port, password)
		sni, _ := p["servername"].(string)
		if sni == "" {
			sni, _ = p["sni"].(string)
		}
		if sni != "" {
			line += ", sni=" + sni
		}
		return line

	case "socks5":
		line := fmt.Sprintf("%s = socks5, %s, %s", name, server, port)
		username, _ := p["username"].(string)
		password, _ := p["password"].(string)
		if username != "" && password != "" {
			line += fmt.Sprintf(", username=%s, password=%s", username, password)
		}
		return line

	default:
		return fmt.Sprintf("# unsupported: %s", name)
	}
}

// quantumultxProxyLine returns a single QuantumultX proxy line for the given proxy map.
func quantumultxProxyLine(p map[string]interface{}) string {
	name, _ := p["name"].(string)
	server, _ := p["server"].(string)
	port := portToString(p["port"])
	proxyType := strings.ToLower(fmt.Sprintf("%v", p["type"]))

	switch proxyType {
	case "ss":
		cipher, _ := p["cipher"].(string)
		password, _ := p["password"].(string)
		return fmt.Sprintf("shadowsocks=%s:%s, method=%s, password=%s, fast-open=false, udp-relay=false, tag=%s",
			server, port, cipher, password, name)

	case "vmess":
		uuid, _ := p["uuid"].(string)
		line := fmt.Sprintf("vmess=%s:%s, method=none, password=%s, fast-open=false, udp-relay=false",
			server, port, uuid)
		tls, _ := p["tls"].(bool)
		net, _ := p["network"].(string)
		if tls && net == "ws" {
			wsPath := ""
			if wsOpts, ok := p["ws-opts"].(map[string]interface{}); ok {
				wsPath, _ = wsOpts["path"].(string)
			}
			line += fmt.Sprintf(", obfs=wss, obfs-uri=%s", wsPath)
		} else if net == "ws" {
			wsPath := ""
			if wsOpts, ok := p["ws-opts"].(map[string]interface{}); ok {
				wsPath, _ = wsOpts["path"].(string)
			}
			line += fmt.Sprintf(", obfs=ws, obfs-uri=%s", wsPath)
		}
		line += fmt.Sprintf(", tag=%s", name)
		return line

	case "trojan":
		password, _ := p["password"].(string)
		return fmt.Sprintf("trojan=%s:%s, password=%s, over-tls=true, tls-verification=true, fast-open=false, udp-relay=false, tag=%s",
			server, port, password, name)

	default:
		return fmt.Sprintf("# unsupported: %s", name)
	}
}

// loonProxyLine returns a single Loon proxy line for the given proxy map.
func loonProxyLine(p map[string]interface{}) string {
	name, _ := p["name"].(string)
	server, _ := p["server"].(string)
	port := portToString(p["port"])
	proxyType := strings.ToLower(fmt.Sprintf("%v", p["type"]))

	switch proxyType {
	case "ss":
		cipher, _ := p["cipher"].(string)
		password, _ := p["password"].(string)
		return fmt.Sprintf("%s = Shadowsocks, %s, %s, %s, %s, fast-open=false, udp=false",
			name, server, port, cipher, password)

	case "vmess":
		uuid, _ := p["uuid"].(string)
		method, _ := p["cipher"].(string)
		if method == "" {
			method = "none"
		}
		return fmt.Sprintf("%s = VMESS, %s, %s, %s, %s, fast-open=false, udp=false",
			name, server, port, method, uuid)

	case "trojan":
		password, _ := p["password"].(string)
		return fmt.Sprintf("%s = Trojan, %s, %s, %s, fast-open=false, udp=false",
			name, server, port, password)

	default:
		return fmt.Sprintf("# unsupported: %s", name)
	}
}

// singboxOutbound converts a proxy map to a sing-box outbound map.
// Returns nil for unrecognized proxy types.
func singboxOutbound(p map[string]interface{}) map[string]interface{} {
	name, _ := p["name"].(string)
	server, _ := p["server"].(string)
	portInt := portToInt(p["port"])
	proxyType := strings.ToLower(fmt.Sprintf("%v", p["type"]))

	switch proxyType {
	case "ss":
		cipher, _ := p["cipher"].(string)
		password, _ := p["password"].(string)
		return map[string]interface{}{
			"type":        "shadowsocks",
			"tag":         name,
			"server":      server,
			"server_port": portInt,
			"method":      cipher,
			"password":    password,
		}

	case "vmess":
		uuid, _ := p["uuid"].(string)
		security, _ := p["cipher"].(string)
		if security == "" {
			security = "auto"
		}
		alterID := 0
		if aid, ok := p["alterId"].(float64); ok {
			alterID = int(aid)
		}
		return map[string]interface{}{
			"type":        "vmess",
			"tag":         name,
			"server":      server,
			"server_port": portInt,
			"uuid":        uuid,
			"alter_id":    alterID,
			"security":    security,
		}

	case "trojan":
		password, _ := p["password"].(string)
		ob := map[string]interface{}{
			"type":        "trojan",
			"tag":         name,
			"server":      server,
			"server_port": portInt,
			"password":    password,
		}
		// Add TLS block if TLS-related fields are present.
		hasTLS := false
		if tls, ok := p["tls"].(bool); ok && tls {
			hasTLS = true
		}
		if sni, ok := p["servername"].(string); ok && sni != "" {
			hasTLS = true
		}
		if hasTLS {
			ob["tls"] = map[string]interface{}{"enabled": true}
		}
		return ob

	case "vless":
		uuid, _ := p["uuid"].(string)
		return map[string]interface{}{
			"type":        "vless",
			"tag":         name,
			"server":      server,
			"server_port": portInt,
			"uuid":        uuid,
		}

	case "socks5":
		return map[string]interface{}{
			"type":        "socks",
			"tag":         name,
			"server":      server,
			"server_port": portInt,
		}

	default:
		return nil
	}
}

// portToInt converts a port value (int, float64, or string) to int.
func portToInt(port interface{}) int {
	switch v := port.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprintf("%v", v))
		return n
	}
}
