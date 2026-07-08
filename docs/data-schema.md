# 🧬 Data Contract — Schema v1

Every HiveScope telemetry message (real-time and replay) uses this JSON schema.
The `schema_version` field allows the format to evolve without breaking compatibility.

## Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | int | ✅ | Schema version. Current: `1`. |
| `timestamp` | string (ISO 8601, UTC) | ✅ | Exact moment of the event. |
| `source` | string | ✅ | Role of the emitting agent (e.g. `"CEO"`, `"Programmer"`). |
| `target` | string \| null | ✅ | Role of the receiving agent. `null` for self-events. |
| `phase` | string | ✅ | Workflow phase (e.g. `"Design"`, `"Coding"`, `"Review"`). |
| `action` | string | ✅ | Short description of the current action. |
| `message` | string | ❌ | Message content exchanged between agents. |
| `files` | string[] | ❌ | Files created or modified in this event. |
| `code` | string | ❌ | Associated code snippet (for the IDE panel). |
| `status` | enum | ✅ | `"idle"` \| `"working"` \| `"done"` \| `"error"`. |

## Example

```json
{
  "schema_version": 1,
  "timestamp": "2026-07-08T14:32:00Z",
  "source": "Programmer",
  "target": "Code Reviewer",
  "phase": "Coding",
  "action": "Writing main.py",
  "message": "Implemented the main game loop.",
  "files": ["main.py"],
  "status": "working"
}
```

## Rules

1. No field may contain personal data from the end user.
2. Adapters (ChatDev, CrewAI, etc.) translate their native format into this schema.
3. Breaking changes require incrementing `schema_version`.
