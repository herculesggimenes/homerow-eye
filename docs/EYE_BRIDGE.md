# Eye Bridge

The eye bridge lets Neru use the gaze recorder's second cursor as an OS-wide
navigation target. Gaze moves the mouse; clicking remains outside the eye
bridge and should stay explicit through normal mouse or keyboard input.

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

Start Neru from this fork. The daemon reads the gaze recorder cursor directly:

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

The same daemon-side target can be used through the normal action command:

```bash
bin/neru action move_mouse --eye
```

## Safety Defaults

- The eye bridge does not click.
- Cursor samples older than `5s` are rejected.
- `eye cursor`, `eye move`, and `action move_mouse --eye` all resolve inside
  the Neru daemon.
