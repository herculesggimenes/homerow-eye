package cli

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/y3owk1n/neru/internal/core/domain/action"
	derrors "github.com/y3owk1n/neru/internal/core/errors"
)

const (
	eyeCursorFileEnv = "GAZE_RECORDER_CURSOR_FILE"
	eyeDataDirEnv    = "GAZE_RECORDER_DATA_DIR"
	eyeDefaultMaxAge = 5 * time.Second
)

// EyeCmd groups commands that target the cursor emitted by the gaze recorder.
var EyeCmd = &cobra.Command{
	Use:   "eye",
	Short: "Use the gaze recorder cursor as a target",
	Long: `Use the cursor stream from the gaze recorder as a Neru action target.

By default, Neru reads ~/src/gaze-mouse-recorder/data/latest-cursor.json.
Set GAZE_RECORDER_DATA_DIR, GAZE_RECORDER_CURSOR_FILE, or --cursor-file to
point at another recorder data directory.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

type eyeCursorPayload struct {
	SessionID string  `json:"sessionId"`
	TS        float64 `json:"ts"`
	ScreenX   float64 `json:"screenX"`
	ScreenY   float64 `json:"screenY"`
	Recording bool    `json:"recording"`
	PageURL   string  `json:"pageUrl"`
}

type eyeCursorTarget struct {
	Payload eyeCursorPayload
	Point   image.Point
	Age     time.Duration
}

func init() {
	EyeCmd.AddCommand(buildEyeCursorCommand())
	EyeCmd.AddCommand(buildEyeMoveCommand())
	EyeCmd.AddCommand(buildEyeClickCommand())

	RootCmd.AddCommand(EyeCmd)
}

func buildEyeCursorCommand() *cobra.Command {
	var cursorFile string
	var maxAge time.Duration

	cmd := &cobra.Command{
		Use:   "cursor",
		Short: "Print the latest gaze cursor target",
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := readEyeCursor(cursorFile, maxAge, time.Now())
			if err != nil {
				return err
			}

			cmd.Printf(
				"x=%d y=%d age=%s session=%s\n",
				target.Point.X,
				target.Point.Y,
				target.Age.Round(time.Millisecond),
				target.Payload.SessionID,
			)

			return nil
		},
	}

	addEyeCursorFlags(cmd, &cursorFile, &maxAge)

	return cmd
}

func buildEyeMoveCommand() *cobra.Command {
	var cursorFile string
	var maxAge time.Duration

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Move the real cursor to the latest gaze target",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := readEyeCursor(cursorFile, maxAge, time.Now())
			if err != nil {
				return err
			}

			return sendEyeMove(cmd, target.Point)
		},
	}

	addEyeCursorFlags(cmd, &cursorFile, &maxAge)

	return cmd
}

func buildEyeClickCommand() *cobra.Command {
	var cursorFile string
	var maxAge time.Duration
	var clickAction string
	var modifier string
	var confirm bool

	cmd := &cobra.Command{
		Use:   "click",
		Short: "Click at the latest gaze target",
		Long: `Click at the latest gaze target.

This command requires --confirm so accidental gaze samples cannot trigger an
OS-wide click from a script or partial command.`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return derrors.New(
					derrors.CodeInvalidInput,
					"eye click requires --confirm",
				)
			}

			if !isEyeClickAction(clickAction) {
				return derrors.Newf(
					derrors.CodeInvalidInput,
					"unsupported eye click action %q",
					clickAction,
				)
			}

			target, err := readEyeCursor(cursorFile, maxAge, time.Now())
			if err != nil {
				return err
			}

			if err := sendEyeMove(cmd, target.Point); err != nil {
				return err
			}

			args := []string{clickAction, "--bare"}
			if modifier != "" {
				args = append(args, "--modifier="+modifier)
			}

			return sendCommand(cmd, "action", args)
		},
	}

	addEyeCursorFlags(cmd, &cursorFile, &maxAge)
	cmd.Flags().StringVar(
		&clickAction,
		"action",
		string(action.NameLeftClick),
		"Mouse action to perform: left_click, right_click, middle_click, mouse_down, or mouse_up",
	)
	cmd.Flags().StringVar(
		&modifier,
		"modifier",
		"",
		"Comma-separated modifier keys to hold during click (cmd, shift, alt, option, ctrl)",
	)
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Required to perform an OS-wide click")

	return cmd
}

func addEyeCursorFlags(cmd *cobra.Command, cursorFile *string, maxAge *time.Duration) {
	cmd.Flags().StringVar(
		cursorFile,
		"cursor-file",
		defaultEyeCursorFile(),
		"Path to latest-cursor.json from the gaze recorder",
	)
	cmd.Flags().DurationVar(
		maxAge,
		"max-age",
		eyeDefaultMaxAge,
		"Maximum accepted cursor sample age; set 0 to disable stale-sample rejection",
	)
}

func defaultEyeCursorFile() string {
	if cursorFile := os.Getenv(eyeCursorFileEnv); cursorFile != "" {
		return cursorFile
	}

	if dataDir := os.Getenv(eyeDataDirEnv); dataDir != "" {
		return filepath.Join(dataDir, "latest-cursor.json")
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, "src", "gaze-mouse-recorder", "data", "latest-cursor.json")
	}

	return filepath.Join("data", "latest-cursor.json")
}

func readEyeCursor(path string, maxAge time.Duration, now time.Time) (eyeCursorTarget, error) {
	if path == "" {
		return eyeCursorTarget{}, derrors.New(
			derrors.CodeInvalidInput,
			"cursor file path is required",
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return eyeCursorTarget{}, derrors.Wrap(
			err,
			derrors.CodeInvalidInput,
			"failed to read gaze cursor file",
		)
	}

	var payload eyeCursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return eyeCursorTarget{}, derrors.Wrap(
			err,
			derrors.CodeInvalidInput,
			"failed to parse gaze cursor file",
		)
	}

	if payload.SessionID == "" {
		return eyeCursorTarget{}, derrors.New(
			derrors.CodeInvalidInput,
			"gaze cursor payload is missing sessionId",
		)
	}

	if !isFinite(payload.TS) || !isFinite(payload.ScreenX) || !isFinite(payload.ScreenY) {
		return eyeCursorTarget{}, derrors.New(
			derrors.CodeInvalidInput,
			"gaze cursor payload contains invalid numeric values",
		)
	}

	sampleTime := unixSecondsToTime(payload.TS)
	age := now.Sub(sampleTime)
	if age < 0 {
		age = -age
	}

	if maxAge > 0 && age > maxAge {
		return eyeCursorTarget{}, derrors.Newf(
			derrors.CodeInvalidInput,
			"gaze cursor sample is stale: age %s exceeds %s",
			age.Round(time.Millisecond),
			maxAge,
		)
	}

	return eyeCursorTarget{
		Payload: payload,
		Point: image.Point{
			X: int(math.Round(payload.ScreenX)),
			Y: int(math.Round(payload.ScreenY)),
		},
		Age: age,
	}, nil
}

func unixSecondsToTime(seconds float64) time.Time {
	whole, frac := math.Modf(seconds)

	return time.Unix(int64(whole), int64(frac*float64(time.Second)))
}

func isFinite(value float64) bool {
	return !math.IsInf(value, 0) && !math.IsNaN(value)
}

func sendEyeMove(cmd *cobra.Command, point image.Point) error {
	return sendCommand(
		cmd,
		"action",
		[]string{
			string(action.NameMoveMouse),
			fmt.Sprintf("--x=%d", point.X),
			fmt.Sprintf("--y=%d", point.Y),
		},
	)
}

func isEyeClickAction(name string) bool {
	switch action.Name(name) {
	case action.NameLeftClick,
		action.NameRightClick,
		action.NameMiddleClick,
		action.NameMouseDown,
		action.NameMouseUp:
		return true
	default:
		return false
	}
}
