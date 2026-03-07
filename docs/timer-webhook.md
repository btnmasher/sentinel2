# Timer Webhook

Sentinel supports a webhook-owned timer mode controlled by `TIMER_SOURCE=webhook`.

In this mode:

- Timer reads remain enabled.
- Timer overlays and the timer board continue to work.
- Manual timer creation, editing, canceling, and deleting are disabled.
- Sovereignty campaign sync and structure notification sync are disabled.
- Cleanup of old inactive timers still runs normally.

## Auth

All webhook requests must send:

```http
Authorization: Bearer <TIMERS_WEBHOOK_BEARER_TOKEN>
```

## Identifier Contract

- Create payload `id` is required.
- Update target `id` is required in the request path.
- Update payload `id` is optional, but when present it must match the path id.
- Sentinel stores this in the timer record `webhook_id` field.
- PocketBase record ids remain internal.

## Field Semantics

- Create payload requires `id`, `system_id`, and `expires_at`.
- Update payload is partial; all fields are optional.
- `region_id` is not part of the webhook contract. Sentinel derives region fields from `system_id`.
- `status` is optional. If omitted:
  - create defaults to `active`
  - update leaves the existing status unchanged
- `status: "canceled"` is only needed when the external system wants the timer to remain stored but marked canceled. It is not needed for normal active timers.
- Create requests must still satisfy normal timer validation:
  - a valid timer context for `timer_kind`, `structure_type`, and `stage_label`
  - `planet_id` when the selected structure requires a planet, such as `orbital_skyhook`
  - `moon_id` when the selected structure requires a moon, such as `metenox_moon_drill`
- Display fields are optional when the corresponding IDs are supplied:
  - `planet_name`
  - `moon_name`
  - `owner_corporation_name`
  - `owner_corporation_ticker`
  - `owner_alliance_name`
  - `owner_alliance_ticker`
- Sentinel will try to backfill those display fields automatically:
  - planets and moons from local SDE data
  - corporations and alliances from the local organization cache, falling back to public ESI by ID when needed
- `system_name` and `region_name` are internal derived fields and are not part of the webhook contract.

## Endpoints

### Create

`POST /api/webhooks/timers`

- Creates a timer for a new external `id`.
- Returns `409 Conflict` when the `id` already exists.
- Requests must satisfy normal timer validation.

Example create:

```json
{
  "id": "ext-123",
  "system_id": 30000142,
  "title": "QRF ansiblex armor timer",
  "standing_type": "hostile",
  "timer_kind": "reinforcement",
  "structure_type": "ansiblex_jump_bridge",
  "stage_label": "armor",
  "replacement_action": "alliance_replacement",
  "severity": "high",
  "expires_at": "2026-03-08T02:30:00Z",
  "notes": "Imported from ops planner"
}
```

Example create response:

```json
{
  "operation": "created",
  "id": "ext-123"
}
```

Example create with IDs only for display-backed fields:

```json
{
  "id": "ext-456",
  "system_id": 30000142,
  "timer_kind": "custom",
  "structure_type": "custom",
  "stage_label": "custom",
  "expires_at": "2026-03-08T02:30:00Z",
  "planet_id": 40349466,
  "moon_id": 40349472,
  "owner_corporation_id": 98765432,
  "owner_alliance_id": 99000001
}
```

If Sentinel already has those IDs in its local cache, it will populate the corresponding names and tickers automatically.

### Update

`PATCH /api/webhooks/timers/{id}`

- Applies a partial update to the timer identified by `webhook_id`.
- Returns `404 Not Found` when the timer does not exist.
- If payload `id` is present, it must match `{id}`.
- Existing business-rule validation still applies after merge.

Example partial update:

```json
{
  "severity": "critical",
  "notes": "Escalated after scout update"
}
```

Example update response:

```json
{
  "operation": "updated",
  "id": "ext-123"
}
```

### Delete

`DELETE /api/webhooks/timers/{id}`

- Hard deletes the timer identified by `webhook_id`.
- Deletes are idempotent and return `204 No Content`.

## Schema

The machine-readable schema is published at [timer-webhook.schema.json](/home/terminal/Code/sentinel2/docs/timer-webhook.schema.json).
