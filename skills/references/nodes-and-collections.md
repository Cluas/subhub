# Managing Nodes & Collections

## Collections

Collections group self-managed proxies or rules. Two types: `proxy` and `rule`.

### Create a Collection

```bash
# Proxy collection
curl -s -X POST $SUBHUB_URL/api/collections \
  -H "Content-Type: application/json" \
  -d '{"name": "my-nodes", "content_type": "proxy"}'

# Rule collection
curl -s -X POST $SUBHUB_URL/api/collections \
  -H "Content-Type: application/json" \
  -d '{"name": "ai-domains", "content_type": "rule"}'
```

### List / Get / Delete

```bash
curl -s $SUBHUB_URL/api/collections                    # List all
curl -s $SUBHUB_URL/api/collections/{id}               # Get one
curl -s $SUBHUB_URL/api/collections/{id}/proxies       # List proxies
curl -s $SUBHUB_URL/api/collections/{id}/rules         # List rules
curl -s -X DELETE $SUBHUB_URL/api/collections/{id}     # Delete
```

## Adding Proxy Nodes

Add nodes via `POST /api/proxies` with `collection_id`. Protocol-specific fields go in `config`.

### VLESS (Reality)

```bash
curl -s -X POST $SUBHUB_URL/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "us-node-01",
    "type": "vless",
    "server": "1.2.3.4",
    "port": 443,
    "collection_id": 2,
    "config": {
      "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "network": "tcp",
      "flow": "xtls-rprx-vision",
      "tls": true,
      "udp": true,
      "servername": "www.example.com",
      "reality-opts": {
        "public-key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        "short-id": "abcd1234"
      },
      "client-fingerprint": "chrome"
    }
  }'
```

### Trojan

```bash
curl -s -X POST $SUBHUB_URL/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hk-trojan-01",
    "type": "trojan",
    "server": "hk.example.com",
    "port": 443,
    "collection_id": 2,
    "config": {
      "password": "your-password"
    }
  }'
```

### Shadowsocks

```bash
curl -s -X POST $SUBHUB_URL/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ss-node",
    "type": "ss",
    "server": "ss.example.com",
    "port": 8388,
    "collection_id": 2,
    "config": {
      "cipher": "chacha20-ietf-poly1305",
      "password": "your-password",
      "udp": true
    }
  }'
```

### SOCKS5

```bash
curl -s -X POST $SUBHUB_URL/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "socks-proxy",
    "type": "socks5",
    "server": "10.0.0.1",
    "port": 1080,
    "collection_id": 2,
    "config": {
      "username": "user",
      "password": "pass",
      "udp": true
    }
  }'
```

### Relay Node (with dialer-proxy)

Relay nodes route through a transit proxy-group. Add `dialer-proxy` in config:

```bash
curl -s -X POST $SUBHUB_URL/api/proxies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "relay-us-01",
    "type": "vless",
    "server": "relay.example.com",
    "port": 443,
    "collection_id": 3,
    "config": {
      "dialer-proxy": "Transit Group",
      "uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "network": "tcp",
      "tls": true,
      "servername": "example.com",
      "reality-opts": { "public-key": "xxx" },
      "client-fingerprint": "chrome"
    }
  }'
```

## Adding Rules to Collections

```bash
curl -s -X POST $SUBHUB_URL/api/rules \
  -H "Content-Type: application/json" \
  -d '{
    "type": "DOMAIN-SUFFIX",
    "payload": "google.com",
    "target": "Proxy",
    "collection_id": 5
  }'
```

### Batch Add (shell loop)

```bash
COL_ID=5
for domain in anthropic.com claude.ai openai.com; do
  curl -s -X POST $SUBHUB_URL/api/rules \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"DOMAIN-SUFFIX\",\"payload\":\"$domain\",\"target\":\"US-Proxy\",\"collection_id\":$COL_ID}"
done
```

## Listing and Filtering Nodes

```bash
# All nodes
curl -s "$SUBHUB_URL/api/proxies"

# Filter by subscription
curl -s "$SUBHUB_URL/api/proxies?subscription_id=1"

# Filter by type and alive status
curl -s "$SUBHUB_URL/api/proxies?type=vless&alive=true"

# Filter by collection
curl -s "$SUBHUB_URL/api/proxies?collection_id=2"
```
