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

## Timer Context Rules

Sentinel validates the timer context using the `timer_kind` + `structure_type` + `stage_label` combination.

Accepted `stage_label` values:

- `armor`
- `reinforcement`
- `hull`
- `initial_vulnerability`
- `not_applicable`
- `anchoring`
- `unanchoring`
- `extraction_window`
- `custom`

### `timer_kind = reinforcement`

Allowed stages and why:

- `armor` or `hull` only for dual-stage reinforcement structures:
  - `upwell_citadel_fortizar`
  - `upwell_citadel_keepstar`
  - `upwell_engineering_azbel`
  - `upwell_engineering_sotiyo`
  - `upwell_refinery_tatara`
  - Reason: these structures have separate armor and hull reinforcement milestones.
- `reinforcement` only for single-stage reinforcement structures:
  - `upwell_citadel_astrahus`
  - `upwell_engineering_raitaru`
  - `upwell_refinery_athanor`
  - `ansiblex_jump_bridge`
  - `pharolux_cyno_beacon`
  - `tenebrex_cyno_jammer`
  - `orbital_skyhook`
  - `metenox_moon_drill`
  - `sovereignty_hub`
  - `mercenary_den`
  - `customs_office_poco`
  - `player_owned_starbase`
  - Reason: these structures use a single reinforcement stage.
- `not_applicable` with any structure.
  - Reason: flexibility

### `timer_kind = anchoring`

- `stage_label` must be `anchoring`.
- `structure_type` must be one of:
  - `upwell_citadel_astrahus`
  - `upwell_citadel_fortizar`
  - `upwell_citadel_keepstar`
  - `upwell_engineering_raitaru`
  - `upwell_engineering_azbel`
  - `upwell_engineering_sotiyo`
  - `upwell_refinery_athanor`
  - `upwell_refinery_tatara`
  - `ansiblex_jump_bridge`
  - `pharolux_cyno_beacon`
  - `tenebrex_cyno_jammer`
  - `orbital_skyhook`
  - `metenox_moon_drill`
  - `customs_office_poco`
  - `player_owned_starbase`
  - `custom`
- Reason: anchoring timers are only valid for deployable structures that can be anchored.

### `timer_kind = unanchoring`

- `stage_label` must be `unanchoring`.
- `structure_type` must be one of:
  - `upwell_citadel_astrahus`
  - `upwell_citadel_fortizar`
  - `upwell_citadel_keepstar`
  - `upwell_engineering_raitaru`
  - `upwell_engineering_azbel`
  - `upwell_engineering_sotiyo`
  - `upwell_refinery_athanor`
  - `upwell_refinery_tatara`
  - `ansiblex_jump_bridge`
  - `pharolux_cyno_beacon`
  - `tenebrex_cyno_jammer`
  - `orbital_skyhook`
  - `metenox_moon_drill`
  - `customs_office_poco`
  - `player_owned_starbase`
  - `custom`
- Reason: unanchoring timers are only valid for structures that support unanchoring.

### `timer_kind = extraction`

- `stage_label` must be `extraction_window`.
- `structure_type` must be one of:
  - `upwell_refinery_athanor`
  - `upwell_refinery_tatara`
  - `orbital_skyhook`
  - `metenox_moon_drill`
  - `mercenary_den`
- Reason: extraction windows apply to resource extraction style structures.
- Extra rule: if `structure_type = orbital_skyhook`, `skyhook_fullness_pct` is required (0-100).

### `timer_kind = custom`

- `stage_label` must be `custom`.
- Reason: custom timers use an explicit custom stage to avoid accidental overlap with structured timer semantics.

## Structure-specific extra requirements

- `planet_id` is required for:
  - `orbital_skyhook`
  - `mercenary_den`
  - Reason: these structures are planet-bound.
- `moon_id` is required for:
  - `metenox_moon_drill`
  - Reason: this structure is moon-bound.

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
  "stage_label": "reinforcement",
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
