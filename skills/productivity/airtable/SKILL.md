---
name: airtable
description: "Airtable REST API via curl. Records CRUD, filters, upserts."
version: 1.1.0
author: community
license: MIT
triggers:
  - Airtable base/table/record work
  - Listing Airtable bases, tables, records, fields, views, or schemas
  - Creating, updating, deleting, filtering, sorting, or upserting Airtable records
exclusions:
  - Live OAuth, sync daemons, or workspace-wide base discovery
  - Browser or UI automation for routine REST API work
  - Loading this skill when Airtable credentials are unavailable
review:
  state: reviewed
  source: hermes:c997183f..7e3c8a31
prerequisites:
  credential_groups:
    - any_of: [AIRTABLE_API_KEY, AIRTABLE_PAT]
  commands: [curl]
metadata:
  hermes:
    tags: [Airtable, Productivity, Database, API]
    homepage: https://airtable.com/developers/web/api/introduction
  gormes:
    source: upstream-hermes
    trust_class: [operator, system]
---

# Airtable - Bases, Tables & Records

Work with Airtable's REST API directly through `curl` when the user asks to read or change Airtable bases, tables, fields, views, or records. Use this cookbook only after Gormes has marked the skill available; missing credentials should keep the skill out of prompts.

Do not set up OAuth, start a sync daemon, discover every workspace base in the background, or use browser automation for routine REST API work.

## Setup

1. Use a Personal Access Token from https://airtable.com/create/tokens. Tokens usually start with `pat`.
2. Grant the minimum scopes needed for the task:
   - `data.records:read` to read rows.
   - `data.records:write` to create, update, delete, or upsert rows.
   - `schema.bases:read` to list bases, tables, fields, and views.
3. Store the token in either `AIRTABLE_API_KEY` or `AIRTABLE_PAT`.
4. In the token UI, add each base the token may access. Airtable PAT access is scoped per base.

Legacy `key...` API keys were deprecated in February 2024. Prefer PATs for all new use.

## API Basics

- Endpoint: `https://api.airtable.com/v0`
- Auth header: `Authorization: Bearer $AIRTABLE_TOKEN`
- Object IDs: bases `app...`, tables `tbl...`, records `rec...`, fields `fld...`.
- JSON writes need `Content-Type: application/json`.
- Airtable limits requests to 5 requests per second per base. On `429`, slow down and retry later.

Initialize shell variables before examples:

```bash
AIRTABLE_TOKEN="${AIRTABLE_API_KEY:-$AIRTABLE_PAT}"
AUTH_HEADER="Authorization: Bearer $AIRTABLE_TOKEN"
BASE_ID=appXXXXXXXXXXXXXX
TABLE=Tasks
```

Prefer stable IDs over names when table names contain spaces, names may change, or the workflow is automated.

## Cookbook

### List bases the token can see

```bash
curl -s "https://api.airtable.com/v0/meta/bases" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

This requires `schema.bases:read`. If Airtable returns `403`, ask the user for the `app...` base ID or for a token with schema read access.

### List tables and fields for a base

```bash
curl -s "https://api.airtable.com/v0/meta/bases/$BASE_ID/tables" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

Inspect schema before mutating records. Confirm table IDs, field names, field IDs, select options, and the primary field.

### List records

```bash
curl -s "https://api.airtable.com/v0/$BASE_ID/$TABLE?maxRecords=10" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

List endpoints return at most 100 records per page. If the response includes `offset`, pass it on the next request and repeat until it is absent.

### Get a single record

```bash
curl -s "https://api.airtable.com/v0/$BASE_ID/$TABLE/$RECORD_ID" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

### Filter records

Always URL-encode `filterByFormula` with Python stdlib instead of hand-escaping:

```bash
FORMULA="{Status}='Todo'"
ENC=$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$FORMULA")
curl -s "https://api.airtable.com/v0/$BASE_ID/$TABLE?filterByFormula=$ENC&maxRecords=20" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

Useful formula patterns:

- Exact match: `{Email}='user@example.com'`
- Contains: `FIND('bug', LOWER({Title}))`
- Multiple conditions: `AND({Status}='Todo', {Priority}='High')`
- Not empty: `NOT({Assignee}='')`
- Date comparison: `IS_AFTER({Due}, TODAY())`

### Sort and select fields

```bash
curl -s "https://api.airtable.com/v0/$BASE_ID/$TABLE?sort%5B0%5D%5Bfield%5D=Priority&sort%5B0%5D%5Bdirection%5D=asc&fields%5B%5D=Name&fields%5B%5D=Status" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

Square brackets in query parameters must be URL-encoded as `%5B` and `%5D`.

### Create a record

```bash
curl -s -X POST "https://api.airtable.com/v0/$BASE_ID/$TABLE" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{"fields":{"Name":"New task","Status":"Todo","Priority":"High"}}' | python3 -m json.tool
```

### Create records in batches

```bash
curl -s -X POST "https://api.airtable.com/v0/$BASE_ID/$TABLE" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "typecast": true,
    "records": [
      {"fields": {"Name": "Task A", "Status": "Todo"}},
      {"fields": {"Name": "Task B", "Status": "In progress"}}
    ]
  }' | python3 -m json.tool
```

Batch endpoints are capped at 10 records per request. For larger inserts, loop in batches of 10 and stay below the per-base rate limit.

### Update a record

Use `PATCH` for partial updates. It preserves fields that are not included in the body.

```bash
curl -s -X PATCH "https://api.airtable.com/v0/$BASE_ID/$TABLE/$RECORD_ID" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{"fields":{"Status":"Done"}}' | python3 -m json.tool
```

### Upsert by a merge field

```bash
curl -s -X PATCH "https://api.airtable.com/v0/$BASE_ID/$TABLE" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "performUpsert": {"fieldsToMergeOn": ["Email"]},
    "records": [
      {"fields": {"Email": "user@example.com", "Status": "Active"}}
    ]
  }' | python3 -m json.tool
```

Use upsert for idempotent sync-like tasks. Airtable creates records whose merge-field values are new and patches records whose merge-field values already exist.

### Delete a record

```bash
curl -s -X DELETE "https://api.airtable.com/v0/$BASE_ID/$TABLE/$RECORD_ID" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

### Delete records in batches

```bash
curl -s -X DELETE "https://api.airtable.com/v0/$BASE_ID/$TABLE?records%5B%5D=recXXXXXXXXXXXXXX&records%5B%5D=recYYYYYYYYYYYYYY" \
  -H "$AUTH_HEADER" | python3 -m json.tool
```

Before bulk deletion, restate the filter and record count and ask the user to confirm.

## Pagination

```bash
OFFSET=""
while :; do
  URL="https://api.airtable.com/v0/$BASE_ID/$TABLE?pageSize=100"
  [ -n "$OFFSET" ] && URL="$URL&offset=$OFFSET"
  RESP=$(curl -s "$URL" -H "$AUTH_HEADER")
  echo "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin); [print(r["id"], r.get("fields",{}).get("Name","")) for r in d.get("records",[])]'
  OFFSET=$(echo "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("offset",""))')
  [ -z "$OFFSET" ] && break
done
```

## Safety Boundaries

- Do not run Airtable calls unless credentials are present and the user asks for Airtable work.
- Do not print token values. Refer to `AIRTABLE_API_KEY`, `AIRTABLE_PAT`, or `$AIRTABLE_TOKEN`.
- Do not infer field names from partial record responses. Empty fields are omitted; check schema first.
- Use `PATCH` by default. `PUT` replaces the full record and clears omitted fields.
- If Airtable returns non-2xx JSON, read the structured error code before retrying.
- A `403` for one base while another works usually means the token access list does not include that base.
- Single-select values must already exist unless the request body sets `"typecast": true`.
