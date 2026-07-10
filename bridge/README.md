# HiveScope Bridge

WebSocket server that receives telemetry from multi-agent AI frameworks and streams it to the HiveScope UI.

## Development

```bash
pip install -e ".[dev]"
uvicorn hivescope_bridge.main:app --reload --port 8765
```

## Tests

```bash
pytest
```
