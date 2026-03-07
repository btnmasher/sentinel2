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

- The request payload `id` field is required for create and update.
- Sentinel stores this in the timer record `webhook_id` field.
- PocketBase record ids remain internal.

## Endpoints

### Upsert

`PUT /api/webhooks/timers`

- Creates a timer when `id` is new.
- Applies a partial update when `id` already exists.
- Create requests must still satisfy normal timer validation.

Example create:

```json
{
  "id": "ext-123",
  "title": "QRF ansiblex armor timer",
  "system_id": 30000142,
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

Example partial update:

```json
{
  "id": "ext-123",
  "severity": "critical",
  "notes": "Escalated after scout update"
}
```

Example upsert response:

```json
{
  "operation": "created",
  "id": "ext-123",
  "record_id": "pocketbase-record-id"
}
```

### Delete

`DELETE /api/webhooks/timers/{id}`

- Hard deletes the timer identified by `webhook_id`.
- Deletes are idempotent and return `204 No Content`.

## Schema

The machine-readable schema is published at [timer-webhook.schema.json](/home/terminal/Code/sentinel2/docs/timer-webhook.schema.json).
