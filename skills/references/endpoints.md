# Creating Endpoints

Endpoints serve filtered proxy or rule lists at `/p/{slug}`. They are the building blocks for profiles.

## Create a Proxy Endpoint

```bash
curl -s -X POST $SUBHUB_URL/api/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Foreign Nodes",
    "slug": "foreign-nodes",
    "output_type": "proxy",
    "format": "clash",
    "subscription_id": 1,
    "filters": {
      "groups": ["Foreign Lines"]
    }
  }'
```

## Create a Rule Endpoint

```bash
curl -s -X POST $SUBHUB_URL/api/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GFW Rules",
    "slug": "gfw-rules",
    "output_type": "rule",
    "format": "clash",
    "subscription_id": 1,
    "filters": {
      "groups": ["GFW Traffic"]
    }
  }'
```

## Create from Collection

```bash
curl -s -X POST $SUBHUB_URL/api/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Personal Proxies",
    "slug": "personal",
    "output_type": "proxy",
    "format": "clash",
    "collection_id": 2
  }'
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `slug` | No | URL slug (auto-generated if empty) |
| `output_type` | No | `proxy` (default) or `rule` |
| `format` | No | `clash` (default), `surge`, `quantumultx`, `shadowrocket`, `loon`, `singbox` |
| `subscription_id` | No* | Source subscription |
| `collection_id` | No* | Source collection |
| `filters` | No | Filter criteria |

*One of `subscription_id` or `collection_id` is required. They are mutually exclusive.

## Filter Options

```json
{
  "filters": {
    "groups": ["Group Name"],
    "types": ["vless", "trojan"],
    "regions": ["HK", "JP"],
    "name_contains": "premium",
    "alive_only": true,
    "target": "PROXY"
  }
}
```

## Access Endpoint Content

```bash
# Public URL (no auth required)
curl -s $SUBHUB_URL/p/{slug}

# Preview via API (auth required)
curl -s $SUBHUB_URL/api/endpoints/{id}/preview \
  -H "Authorization: Bearer $TOKEN"
```

## Update Endpoint

```bash
curl -s -X PUT $SUBHUB_URL/api/endpoints/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "new-slug",
    "filters": {"groups": ["New Group"]}
  }'
```

## Delete Endpoint

```bash
curl -s -X DELETE $SUBHUB_URL/api/endpoints/{id}
```
