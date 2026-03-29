# SubHub Skill

Manage proxy subscriptions, nodes, collections, profiles, and endpoints via the SubHub API.

## Config

```yaml
# SubHub server address (default: http://localhost:9000)
SUBHUB_URL: http://localhost:9000
# Optional: Bearer token for protected APIs
SUBHUB_TOKEN: ""
```

## When to use

- User says "add subscription", "create profile", "manage proxies", "subhub"
- User wants to manage Clash/Mihomo proxy configurations
- User needs to add/remove nodes, rules, or collections
- User wants to generate or preview a Clash config

## References

Detailed guides for common operations:

- [Managing Subscriptions](references/subscriptions.md) -- Add, fetch, refresh proxy sources
- [Managing Nodes & Collections](references/nodes-and-collections.md) -- Self-managed nodes, relay nodes, proxy/rule collections
- [Creating Endpoints](references/endpoints.md) -- Filtered proxy/rule provider URLs
- [Building Profiles](references/profiles.md) -- Complete Clash/Mihomo config generation
- [Routing Rules](references/routing-rules.md) -- Domain, GeoSite, GeoIP, RULE-SET routing

## Quick Reference

### Data Model

```
Subscription (Clash/V2Ray/SIP002 URL)
  └── Proxies (auto-fetched nodes)
  └── Rules (auto-fetched rules)

Collection (manual grouping)
  └── Proxies (self-managed nodes)  ← POST /api/proxies with collection_id
  └── Rules (self-managed rules)    ← POST /api/rules with collection_id

Endpoint (serves filtered proxy/rule lists)
  ├── source: Subscription OR Collection
  ├── output_type: proxy | rule
  └── serves at: /p/{slug}

Profile (complete Clash config)
  ├── settings: port, mode, log-level, ...
  ├── Node Pools → reference Endpoints (proxy-providers)
  ├── Strategies → proxy-groups (select/auto/fallback)
  ├── Rule Sets → reference Endpoints or external URLs (rule-providers)
  ├── Routing Rules → ordered rules list
  └── serves at: /profile/{slug}
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/profile/{slug}` | Serve profile as Clash YAML |
| GET | `/p/{slug}` | Serve endpoint proxy/rule list |
| GET/POST | `/api/subscriptions` | Subscription management |
| POST | `/api/subscriptions/{id}/fetch` | Fetch/refresh nodes |
| POST | `/api/subscriptions/{id}/health-check` | Run health check |
| GET/POST | `/api/proxies` | Proxy node management |
| GET/POST | `/api/rules` | Rule management |
| GET/POST | `/api/collections` | Collection management |
| GET | `/api/collections/{id}/proxies` | List collection proxies |
| GET | `/api/collections/{id}/rules` | List collection rules |
| GET/POST | `/api/endpoints` | Endpoint management |
| GET | `/api/endpoints/{id}/preview` | Preview endpoint output |
| GET/POST | `/api/profiles` | Profile management |
| GET/POST | `/api/profiles/{id}/node-pools` | Node pool management |
| GET/POST | `/api/profiles/{id}/strategies` | Strategy management |
| GET/POST | `/api/profiles/{id}/rule-sets` | Rule set management |
| GET/POST | `/api/profiles/{id}/routing-rules` | Routing rule management |
| GET/PUT | `/api/settings` | System settings |
| GET | `/api/dashboard/stats` | Dashboard statistics |
| GET | `/healthz` | Health check |

### Supported Types

**Proxy types:** `ss`, `ssr`, `vmess`, `vless`, `trojan`, `hysteria2`, `socks5`

**Rule types:** `DOMAIN`, `DOMAIN-SUFFIX`, `DOMAIN-KEYWORD`, `IP-CIDR`, `GEOIP`, `GEOSITE`, `RULE-SET`, `MATCH`

**Strategy types:** `select`, `auto` (url-test), `fallback`, `load_balance`

**Output formats:** `clash`, `surge`, `quantumultx`, `shadowrocket`, `loon`, `singbox`
