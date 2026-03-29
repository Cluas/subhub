# Managing Subscriptions

## Add a Subscription

```bash
curl -s -X POST $SUBHUB_URL/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-provider",
    "url": "https://example.com/subscribe?type=clash",
    "type": "clash",
    "auto_refresh": true,
    "refresh_cron": "0 */6 * * *"
  }'
```

**Fields:**
- `name` (required) -- Display name
- `url` (required) -- Subscription URL
- `type` (required) -- `clash`, `v2ray`, or `sip002`
- `auto_refresh` -- Enable cron-based refresh
- `refresh_cron` -- Cron expression (e.g. `0 */6 * * *` = every 6 hours)

## Fetch Nodes

After creating, fetch to pull nodes:

```bash
curl -s -X POST $SUBHUB_URL/api/subscriptions/{id}/fetch
```

Returns the updated subscription with `node_count`.

## Update Subscription

```bash
curl -s -X PUT $SUBHUB_URL/api/subscriptions/{id} \
  -H "Content-Type: application/json" \
  -d '{"name": "new-name", "url": "https://new-url.com/sub"}'
```

## List Subscriptions

```bash
curl -s $SUBHUB_URL/api/subscriptions
```

## Delete Subscription

Deletes the subscription and all its fetched nodes/rules:

```bash
curl -s -X DELETE $SUBHUB_URL/api/subscriptions/{id}
```

## Health Check

Check liveness of all nodes in a subscription:

```bash
curl -s -X POST $SUBHUB_URL/api/subscriptions/{id}/health-check
```

Returns `{ "checked": N, "alive": M }`.
