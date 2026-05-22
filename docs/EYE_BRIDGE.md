# Eye Bridge

The eye bridge lets Neru use the gaze recorder's second cursor as an OS-wide
action target. This is the first step toward a Homerow-like workflow where gaze
narrows the target and an explicit keyboard action confirms the click.

## Source

By default, Neru reads:

```bash
~/src/gaze-mouse-recorder/data/latest-cursor.json
```

Override that with either:

```bash
GAZE_RECORDER_DATA_DIR=/path/to/data
GAZE_RECORDER_CURSOR_FILE=/path/to/latest-cursor.json
```

or pass `--cursor-file`.

## Usage

Start the gaze recorder pieces first:

```bash
cd ~/src/gaze-mouse-recorder
pnpm collector
pnpm dev
pnpm companion
```

Start Neru from this fork:

```bash
cd ~/src/neru-eye
CGO_ENABLED=1 go build -o bin/neru ./cmd/neru
bin/neru launch
```

Inspect the latest gaze target without moving the real cursor:

```bash
bin/neru eye cursor
```

Move the real cursor to the latest gaze target:

```bash
bin/neru eye move
```

Click at the latest gaze target:

```bash
bin/neru eye click --confirm
```

Right-click and modifier clicks are also supported:

```bash
bin/neru eye click --confirm --action right_click
bin/neru eye click --confirm --modifier cmd
```

## Safety Defaults

- `eye click` requires `--confirm`.
- Cursor samples older than `5s` are rejected.
- Use `--max-age 0` only for debugging stale data.
- `eye cursor` is read-only and does not require the Neru daemon.
