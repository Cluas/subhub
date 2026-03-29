package engine

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ProxyToLink converts a map[string]interface{} proxy config into a subscription link.
// Supports vless, trojan, vmess, socks5.
func ProxyToLink(p map[string]interface{}) string {
	proxyType, _ := p["type"].(string)
	switch strings.ToLower(proxyType) {
	case "vless":
		return vlessToLink(p)
	case "trojan":
		return trojanToLink(p)
	case "vmess":
		return vmessToLink(p)
	case "socks5":
		return socks5ToLink(p)
	}
	return ""
}

// ProxiesToLinks batch-converts proxies and returns Base64-encoded subscription content.
func ProxiesToLinks(proxies []map[string]interface{}) string {
	var links []string
	for _, p := range proxies {
		if link := ProxyToLink(p); link != "" {
			links = append(links, link)
		}
	}
	content := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content))
}

func vlessToLink(p map[string]interface{}) string {
	uuid, _ := p["uuid"].(string)
	server, _ := p["server"].(string)
	if uuid == "" || server == "" {
		return ""
	}

	port := portToString(p["port"])
	name, _ := p["name"].(string)
	params := url.Values{}

	network, _ := p["network"].(string)
	if network == "" {
		network = "tcp"
	}
	params.Set("type", network)

	realityOpts, hasReality := p["reality-opts"].(map[string]interface{})
	tls, _ := p["tls"].(bool)
	servername, _ := p["servername"].(string)

	if hasReality && len(realityOpts) > 0 {
		params.Set("security", "reality")
		if servername != "" {
			params.Set("sni", servername)
		}
		if pk, ok := realityOpts["public-key"].(string); ok && pk != "" {
			params.Set("pbk", pk)
		}
		if sid, ok := realityOpts["short-id"].(string); ok && sid != "" {
			params.Set("sid", sid)
		}
	} else if tls {
		params.Set("security", "tls")
		if servername != "" {
			params.Set("sni", servername)
		}
	}

	if flow, ok := p["flow"].(string); ok && flow != "" {
		params.Set("flow", flow)
	}
	if fp, ok := p["client-fingerprint"].(string); ok && fp != "" {
		params.Set("fp", fp)
	}

	if network == "ws" {
		if wsOpts, ok := p["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok && path != "" {
				params.Set("path", path)
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok && host != "" {
					params.Set("host", host)
				}
			}
		}
	}

	return fmt.Sprintf("vless://%s@%s:%s?%s#%s", uuid, server, port, params.Encode(), url.QueryEscape(name))
}

func trojanToLink(p map[string]interface{}) string {
	password, _ := p["password"].(string)
	server, _ := p["server"].(string)
	if password == "" || server == "" {
		return ""
	}

	port := portToString(p["port"])
	name, _ := p["name"].(string)
	params := url.Values{}

	sni, _ := p["servername"].(string)
	if sni == "" {
		sni, _ = p["sni"].(string)
	}
	if sni != "" {
		params.Set("sni", sni)
	}
	if skip, _ := p["skip-cert-verify"].(bool); skip {
		params.Set("allowInsecure", "1")
	}

	query := params.Encode()
	if query != "" {
		return fmt.Sprintf("trojan://%s@%s:%s?%s#%s", url.QueryEscape(password), server, port, query, url.QueryEscape(name))
	}
	return fmt.Sprintf("trojan://%s@%s:%s#%s", url.QueryEscape(password), server, port, url.QueryEscape(name))
}

func socks5ToLink(p map[string]interface{}) string {
	server, _ := p["server"].(string)
	if server == "" {
		return ""
	}

	port := portToString(p["port"])
	name, _ := p["name"].(string)
	username, _ := p["username"].(string)
	password, _ := p["password"].(string)

	var auth string
	if username != "" && password != "" {
		auth = fmt.Sprintf("%s:%s@", url.QueryEscape(username), url.QueryEscape(password))
	}
	return fmt.Sprintf("socks5://%s%s:%s#%s", auth, server, port, url.QueryEscape(name))
}

func vmessToLink(p map[string]interface{}) string {
	uuid, _ := p["uuid"].(string)
	server, _ := p["server"].(string)
	if uuid == "" || server == "" {
		return ""
	}

	port := portToString(p["port"])
	portInt, _ := strconv.Atoi(port)
	name, _ := p["name"].(string)
	network, _ := p["network"].(string)
	if network == "" {
		network = "tcp"
	}
	cipher, _ := p["cipher"].(string)
	if cipher == "" {
		cipher = "auto"
	}
	alterID, _ := p["alterId"].(float64)
	tls, _ := p["tls"].(bool)

	cfg := map[string]interface{}{
		"v":    "2",
		"ps":   name,
		"add":  server,
		"port": portInt,
		"id":   uuid,
		"aid":  int(alterID),
		"scy":  cipher,
		"net":  network,
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
		"sni":  "",
	}

	if tls {
		cfg["tls"] = "tls"
		sni, _ := p["servername"].(string)
		if sni == "" {
			sni, _ = p["sni"].(string)
		}
		if sni != "" {
			cfg["sni"] = sni
		}
	}

	if network == "ws" {
		if wsOpts, ok := p["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok {
				cfg["path"] = path
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					cfg["host"] = host
				}
			}
		}
	}

	data, _ := json.Marshal(cfg)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func portToString(port interface{}) string {
	switch v := port.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.Itoa(int(v))
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
