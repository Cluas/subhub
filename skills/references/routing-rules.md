# Routing Rules

Routing rules determine how traffic is routed in the Clash config. They are ordered by `position` -- first match wins.

## Rule Types

| Type | Syntax | Example |
|------|--------|---------|
| `DOMAIN` | Exact domain | `google.com` |
| `DOMAIN-SUFFIX` | Domain suffix | `google.com` matches `*.google.com` |
| `DOMAIN-KEYWORD` | Keyword in domain | `google` matches any domain containing "google" |
| `IP-CIDR` | IP range | `91.108.56.0/22` |
| `GEOIP` | Country code | `CN`, `US`, `JP` |
| `GEOSITE` | GeoSite category | `gfw`, `cn`, `google`, `netflix` |
| `RULE-SET` | Rule provider ref | References a rule-set by name |
| `MATCH` | Catch-all | Always matches (must be last) |

## API Format

```bash
curl -s -X POST $SUBHUB_URL/api/profiles/{id}/routing-rules \
  -H "Content-Type: application/json" \
  -d '{
    "match": "DOMAIN-SUFFIX",
    "value": "google.com",
    "target": "Proxy Group Name",
    "no_resolve": false,
    "position": 10
  }'
```

**Fields:**
- `match` (required) -- Rule type
- `value` -- Match value (empty for MATCH)
- `target` (required) -- Proxy group name or `DIRECT`/`REJECT`
- `no_resolve` -- Skip DNS resolution for IP rules (default: false)
- `position` (required) -- Order priority (lower = higher priority)

## Common GeoSite Categories

| Category | Description |
|----------|-------------|
| `gfw` | Great Firewall blocked sites |
| `cn` | China domestic sites |
| `google` | Google services |
| `netflix` | Netflix |
| `youtube` | YouTube |
| `openai` | OpenAI/ChatGPT |
| `category-ai-!cn` | AI services (non-China) |
| `category-social-media-!cn` | Social media (non-China) |
| `apple` | Apple services |
| `microsoft` | Microsoft services |
| `steam` | Steam gaming |
| `private` | Private/LAN addresses |
| `category-games` | Gaming sites |
| `category-entertainment` | Entertainment |

## Typical Rule Order

```
1.  RULE-SET  company-internal  → Company VPN
2.  RULE-SET  whitelist         → Whitelist Group
3.  RULE-SET  direct-domains    → DIRECT
4.  RULE-SET  us-exit-domains   → US Proxy
5.  RULE-SET  proxy-domains     → Node Select
10. GEOSITE   openai            → Node Select
11. GEOSITE   google            → US Proxy
12. GEOSITE   youtube           → Node Select
20. GEOSITE   apple             → DIRECT
30. GEOSITE   gfw               → Node Select
40. GEOIP     telegram          → Node Select (no-resolve)
50. GEOSITE   cn                → DIRECT
51. GEOIP     cn                → DIRECT (no-resolve)
99. MATCH                       → Fallback
```
