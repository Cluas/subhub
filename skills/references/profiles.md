# Building Profiles

A Profile composes everything into a complete Clash/Mihomo configuration served at `/profile/{slug}`.

## Create a Profile

```bash
curl -s -X POST $SUBHUB_URL/api/profiles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Driver",
    "settings": {
      "port": 7890,
      "socks-port": 7891,
      "allow-lan": true,
      "mode": "rule",
      "log-level": "info",
      "external-controller": "127.0.0.1:9090"
    }
  }'
```

The response includes `id` and `slug`. The profile serves at `/profile/{slug}`.

## Update Profile (name, slug, settings)

```bash
curl -s -X PUT $SUBHUB_URL/api/profiles/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Name",
    "slug": "daily",
    "settings": {"port": 7890, "mode": "rule"}
  }'
```

Slug must be unique (returns 409 if taken).

## Add Node Pools (proxy-providers)

Node pools reference endpoints. They become `proxy-providers` in the Clash config.

```bash
PROFILE_ID=1

curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/node-pools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "foreign",
    "endpoint_id": 1,
    "endpoint_slug": "foreign-nodes",
    "health_check_url": "http://www.gstatic.com/generate_204",
    "health_check_interval": 300,
    "position": 1
  }'
```

## Add Strategies (proxy-groups)

Strategies define how nodes are selected. They become `proxy-groups` in Clash.

### Select (manual selection)

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/strategies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Node Select",
    "strategy": "select",
    "pools": ["foreign", "personal"],
    "proxies": [],
    "config": {
      "include-all": true,
      "filter": ".*keyword.*"
    },
    "position": 1
  }'
```

### URL-Test (auto lowest latency)

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/strategies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Auto Select",
    "strategy": "auto",
    "pools": ["foreign"],
    "config": {
      "url": "http://cp.cloudflare.com/generate_204",
      "interval": 60,
      "tolerance": 20,
      "include-all": true,
      "filter": ".*premium.*"
    },
    "position": 2
  }'
```

### Fallback

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/strategies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Fallback",
    "strategy": "fallback",
    "pools": ["personal"],
    "config": {
      "include-all": true,
      "interval": 300,
      "filter": ".*tailscale.*"
    },
    "position": 3
  }'
```

### Static Proxy References

Strategies can reference other groups or special keywords by name:

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/strategies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Catch All",
    "strategy": "select",
    "pools": [],
    "proxies": ["DIRECT", "Node Select", "Auto Select"],
    "config": {},
    "position": 4
  }'
```

### Config Options

| Key | Type | Description |
|-----|------|-------------|
| `url` | string | Health check URL |
| `interval` | int | Check interval (seconds) |
| `tolerance` | int | Latency tolerance (ms) |
| `include-all` | bool | Include all nodes from pools |
| `filter` | string | Regex filter for node names |

## Add Rule Sets (rule-providers)

### From SubHub Endpoint

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/rule-sets \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gfw-rules",
    "endpoint_id": 6,
    "endpoint_slug": "gfw-rules",
    "metadata": {"behavior": "classical"},
    "interval": 86400,
    "position": 1
  }'
```

### From External URL

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/rule-sets \
  -H "Content-Type: application/json" \
  -d '{
    "name": "apple",
    "url": "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/apple.txt",
    "metadata": {"behavior": "domain"},
    "interval": 86400,
    "position": 2
  }'
```

### Metadata Behaviors

| Behavior | Description |
|----------|-------------|
| `classical` | Mixed rule types (DOMAIN, IP-CIDR, etc.) |
| `domain` | Domain-only rules |
| `ipcidr` | IP CIDR-only rules |

## Add Routing Rules

Rules are ordered by `position`. First match wins.

```bash
# Domain rule
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/routing-rules \
  -H "Content-Type: application/json" \
  -d '{"match": "DOMAIN-SUFFIX", "value": "google.com", "target": "US Proxy", "position": 1}'

# Rule set reference
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/routing-rules \
  -H "Content-Type: application/json" \
  -d '{"match": "RULE-SET", "value": "gfw-rules", "target": "Node Select", "position": 10}'

# GeoSite
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/routing-rules \
  -H "Content-Type: application/json" \
  -d '{"match": "GEOSITE", "value": "cn", "target": "DIRECT", "position": 20}'

# GeoIP with no-resolve
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/routing-rules \
  -H "Content-Type: application/json" \
  -d '{"match": "GEOIP", "value": "cn", "target": "DIRECT", "no_resolve": true, "position": 21}'

# Catch-all (must be last)
curl -s -X POST $SUBHUB_URL/api/profiles/$PROFILE_ID/routing-rules \
  -H "Content-Type: application/json" \
  -d '{"match": "MATCH", "value": "", "target": "Catch All", "position": 999}'
```

## Access Profile Config

```bash
# Clash YAML output (public, no auth)
curl -s $SUBHUB_URL/profile/{slug}
```

Use this URL as your Clash/Mihomo subscription URL.

## Complete Example

```bash
SUBHUB_URL=http://localhost:9000

# 1. Create subscription and fetch
SUB=$(curl -s -X POST $SUBHUB_URL/api/subscriptions -H "Content-Type: application/json" \
  -d '{"name":"provider","url":"https://sub.example.com/clash","type":"clash"}')
SUB_ID=$(echo $SUB | jq .id)
curl -s -X POST $SUBHUB_URL/api/subscriptions/$SUB_ID/fetch

# 2. Create proxy endpoint
EP=$(curl -s -X POST $SUBHUB_URL/api/endpoints -H "Content-Type: application/json" \
  -d "{\"name\":\"all\",\"slug\":\"all\",\"output_type\":\"proxy\",\"format\":\"clash\",\"subscription_id\":$SUB_ID}")
EP_ID=$(echo $EP | jq .id)

# 3. Create profile
PROF=$(curl -s -X POST $SUBHUB_URL/api/profiles -H "Content-Type: application/json" \
  -d '{"name":"main","settings":{"port":7890,"mode":"rule"}}')
PROF_ID=$(echo $PROF | jq .id)
PROF_SLUG=$(echo $PROF | jq -r .slug)

# 4. Add node pool
curl -s -X POST $SUBHUB_URL/api/profiles/$PROF_ID/node-pools -H "Content-Type: application/json" \
  -d "{\"name\":\"main\",\"endpoint_id\":$EP_ID,\"endpoint_slug\":\"all\",\"health_check_url\":\"http://www.gstatic.com/generate_204\",\"health_check_interval\":300}"

# 5. Add strategy
curl -s -X POST $SUBHUB_URL/api/profiles/$PROF_ID/strategies -H "Content-Type: application/json" \
  -d '{"name":"Auto","strategy":"auto","pools":["main"],"config":{"url":"http://cp.cloudflare.com/generate_204","interval":60}}'

# 6. Add routing rules
curl -s -X POST $SUBHUB_URL/api/profiles/$PROF_ID/routing-rules -H "Content-Type: application/json" \
  -d '{"match":"GEOSITE","value":"cn","target":"DIRECT","position":1}'
curl -s -X POST $SUBHUB_URL/api/profiles/$PROF_ID/routing-rules -H "Content-Type: application/json" \
  -d '{"match":"MATCH","value":"","target":"Auto","position":999}'

# 7. Use it
echo "Subscribe URL: $SUBHUB_URL/profile/$PROF_SLUG"
```
