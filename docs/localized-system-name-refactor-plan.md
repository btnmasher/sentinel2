# Localized System Name Refactor Plan

## Goal

Replace the current `showinfo`-based system resolution path with locale-aware system name matching backed by SDE-imported localized names. The report data should still store and display the canonical English system name, but validation should accept any localized SDE name and preserve the system ID as the primary identity.

## Scope

- Revert the report-submission `showinfo` parsing changes.
- Extend the SDE system import to ingest the supported localized names from `mapSolarSystems.jsonl`.
- Persist one field per supported locale on `solar_systems`.
- Add indexes for each localized name field.
- Update report validation to resolve names in a prioritized order:
  - English first.
  - Then any localized fields that are likely to matter for the submitted text.
- Keep `system_id` as the canonical identifier in report data.

## Proposed Data Model

Add localized name columns to `solar_systems`, likely as text fields such as:

- `name_ja`
- `name_ko`
- `name_zh`

The exact locale set should match the keys emitted by the SDE JSONL `name` object. If the upstream SDE adds or removes locales later, the importer should fail safe by ignoring unknown locales until the migration is updated.

## Import Changes

1. Update the solar-system JSONL parser to read the localized `name` object instead of a single string.
2. Continue populating the canonical `name` field from English.
3. Persist only the supported non-English locale variants (`ja`, `ko`, `zh`) into their corresponding `name_<locale>` fields.
4. Continue caching the canonical English name in memory for existing import consumers.
5. Keep existing validation rules for system ID, region, constellation, and coordinates.
6. Wire the SDE import job so the localized system-name fields are populated during the regular import run and not via a separate follow-up job.

## Migration Plan

1. Create a new PocketBase migration that adds the localized text fields to `solar_systems`.
2. Add indexes for each localized field so name resolution can query efficiently.
3. Backfill localized names from the next SDE import run.
4. Ensure the migration is idempotent and safe to rerun.
5. Leave the existing `idx_solar_systems_eve_id` index unchanged.

## Lookup Strategy

Implement a resolver that attempts matches in this order:

1. Exact English name match.
2. Exact localized-field matches when the input text contains non-English characters or other strong signals of localization.
3. Optional fallback scans across the remaining localized fields if English and the most likely locale candidates do not match.

Optimization idea:

- Short-circuit localized lookups when the submitted system text is plain ASCII and already looks like a normal English system name.
- Still allow localized lookup when the text contains Cyrillic, CJK characters, accented characters, or other non-ASCII runes.
- Do not rely on punctuation or hyphen counts as a locale signal, because many English and nullsec names use them normally.

## Report Processing Changes

1. Remove `showinfo` normalization from the submission path.
2. Preserve the submitted message text exactly as written by the client.
3. When a localized system name is matched, replace the display text in stored/displayed report data with the canonical English system name.
4. Keep the matched system ID alongside the report so downstream features can still use stable identity data.
5. If a report contains multiple candidate system names, prefer the first valid resolution in the message body.

## Validation Rules

- English system name remains the primary and fastest resolution path.
- Localized name matches are acceptable and should resolve to the same system ID as English.
- Unknown names remain invalid and should continue to produce the same user-facing validation error.
- If multiple systems share the same localized text, prefer the system whose English match or strongest locale match appears first in the report text.

## Testing Plan

1. Add importer tests that verify localized `mapSolarSystems.jsonl` rows populate the supported fields.
2. Add lookup tests for English and non-English system names.
3. Add tests for the ASCII short-circuit optimization.
4. Add report-submission tests that confirm the canonical English system name is stored/displayed even when the submitted text uses a localized name.
5. Add regression tests proving that `showinfo` markup is no longer required and no longer used.

## Rollout Plan

1. Merge the migration first.
2. Deploy the importer and validation changes together so the new columns are populated before localized matching is relied on.
3. Run a full SDE import after deployment to backfill localized system names.
4. Verify a representative set of localized report submissions in staging before promoting to production.

## Risks

- The SDE locale set may differ from what the importer currently expects.
- Index growth on `solar_systems` will increase collection size and import cost.
- Naive locale detection can create false positives if it over-queries localized fields.
- Duplicate localized names across systems could require deterministic tie-breaking.

## Done Criteria

- Report submissions no longer depend on `showinfo` markup.
- Localized system names import correctly.
- Validation resolves both English and localized system names to the same system ID.
- Displayed report text uses the canonical English system name.
- The new indexes keep localized lookup latency acceptable.
