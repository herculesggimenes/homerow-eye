package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/y3owk1n/neru/internal/cli/cliutil"
)

// EyeCmd groups commands that use the daemon-owned eye target.
var EyeCmd = &cobra.Command{
	Use:   "eye",
	Short: "Use the latest eye target as a navigation target",
	Long: `Use the latest eye source target as a Neru navigation target.

The Neru daemon resolves eye targets directly. Configure the current file source
before launching Neru with NERU_EYE_DATA_DIR or NERU_EYE_CURSOR_FILE.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	EyeCmd.AddCommand(buildEyeCursorCommand())
	EyeCmd.AddCommand(buildEyeMoveCommand())

	RootCmd.AddCommand(EyeCmd)
}

func buildEyeCursorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cursor",
		Short: "Print the daemon's latest gaze navigation target",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			communicator := cliutil.NewIPCCommunicator(timeoutSec)
			response, err := communicator.SendCommand("eye", []string{"cursor"})
			if err != nil {
				return err
			}

			if !response.Success {
				return communicator.HandleResponse(cmd, response)
			}

			data, ok := response.Data.(map[string]any)
			if !ok {
				cmd.Println(response.Message)

				return nil
			}

			x, _ := numberFromAny(data["x"])
			y, _ := numberFromAny(data["y"])
			ageMS, _ := numberFromAny(data["age_ms"])
			sessionID, _ := data["session_id"].(string)

			cmd.Printf("x=%.0f y=%.0f age=%.0fms session=%s\n", x, y, ageMS, sessionID)

			return nil
		},
	}
}

func buildEyeMoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "move",
		Short: "Move the real cursor to the latest gaze navigation target",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sendCommand(cmd, "action", []string{"move_mouse", "--eye"})
		},
	}
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		floatValue, err := typed.Float64()

		return floatValue, err == nil
	default:
		return 0, false
	}
}
