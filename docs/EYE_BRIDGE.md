# Eye Navigation

Neru's eye path is navigation-only. It should turn an eye source into a mouse
target; it should not click, type, or confirm actions on its own.

The browser/WebGazer prototype is no longer the product path. The useful
boundary is a small native eye target signal that Neru can consume from the
daemon process.

## Source

Until a native camera or hardware tracker provider is wired in, Neru reads the
latest eye target from:

```bash
~/.local/share/neru/eye/latest-cursor.json
```

Override that with either:

```bash
NERU_EYE_DATA_DIR=/path/to/data
NERU_EYE_CURSOR_FILE=/path/to/latest-cursor.json
```

The file shape is intentionally small:

```json
{
  "sessionId": "native-eye",
  "ts": 1760000000.123,
  "screenX": 800,
  "screenY": 450
}
```

## Usage

Start Neru:

```bash
cd ~/src/neru-eye
CGO_ENABLED=1 go build -o bin/neru ./cmd/neru
bin/neru launch
```

Inspect the latest gaze target without moving the real cursor:

```bash
bin/neru eye cursor
```

Move the real cursor to the latest eye target:

```bash
bin/neru eye move
```

The same daemon-side target can be used through the normal action command:

```bash
bin/neru action move_mouse --eye
```

## Safety Defaults

- The eye path does not click.
- Cursor samples older than `5s` are rejected.
- `eye cursor`, `eye move`, and `action move_mouse --eye` all resolve inside
  the Neru daemon.
