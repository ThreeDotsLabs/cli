package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func registerTools(srv *server.MCPServer, state *LoopState) {
	srv.AddTool(
		mcp.NewTool("training_get_exercise_info",
			mcp.WithDescription("Returns information about the current exercise, its state, and optionally logs from the last execution"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithBoolean("include_logs",
				mcp.Description("If true, includes stdout/stderr logs from the last execution. Disabled by default to save context."),
			),
		),
		handleGetExerciseInfo(state),
	)

	srv.AddTool(
		mcp.NewTool("training_run_solution",
			mcp.WithDescription("Triggers the exercise solution to run. Only works when the interactive loop is waiting at a prompt (failed or succeeded state)."),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleRunSolution(state),
	)

	srv.AddTool(
		mcp.NewTool("training_next_exercise",
			mcp.WithDescription("Advances to the next exercise. Optionally syncs with the example solution first. Only works when the current exercise has been completed successfully."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithBoolean("sync_solution",
				mcp.Description("If true, replaces the student's code with the example solution before advancing (equivalent to pressing 's' in the terminal)."),
			),
		),
		handleNextExercise(state),
	)

	srv.AddTool(
		mcp.NewTool("training_reset_exercise",
			mcp.WithDescription("Resets the current exercise to clean files. Your current code is saved to a git backup branch. Only works when the interactive loop is waiting at a prompt (failed or succeeded state). After reset, the exercise is re-run automatically."),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithReadOnlyHintAnnotation(false),
		),
		handleResetExercise(state),
	)

	srv.AddTool(
		mcp.NewTool("training_send_feedback",
			mcp.WithDescription("Submits user feedback about a training exercise"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("training",
				mcp.Required(),
				mcp.Description("Training name (e.g. 'go-event-driven')"),
			),
			mcp.WithString("exercise",
				mcp.Required(),
				mcp.Description("Exercise ID"),
			),
			mcp.WithString("history",
				mcp.Description("Conversation history (optional, only if the student consented to sharing)"),
			),
			mcp.WithNumber("rating",
				mcp.Required(),
				mcp.Description("Rating value (integer, 0-10 where 0 is terrible and 10 is great)"),
			),
			mcp.WithString("feedback",
				mcp.Required(),
				mcp.Description("Feedback text"),
			),
		),
		handleSendFeedback(state),
	)
}

type exerciseInfoResponse struct {
	ExerciseID         string `json:"exercise_id"`
	Directory          string `json:"directory"`
	IsTextOnly         bool   `json:"is_text_only"`
	IsOptional         bool   `json:"is_optional"`
	ModuleName         string `json:"module_name"`
	ExerciseName       string `json:"exercise_name"`
	State              string `json:"state"`
	PendingAction      string `json:"pending_action,omitempty"`
	Error              string `json:"error,omitempty"`
	Logs               string `json:"logs,omitempty"`
	UpdateAvailable    bool   `json:"update_available,omitempty"`
	UpdateVersion      string `json:"update_version,omitempty"`
	UpdateCommand      string `json:"update_command,omitempty"`
	UpdateReleaseNotes string `json:"update_release_notes,omitempty"`
}

func handleGetExerciseInfo(state *LoopState) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := state.GetExerciseInfo()
		currentState := state.GetState()

		resp := exerciseInfoResponse{
			ExerciseID:    info.ExerciseID,
			Directory:     info.Directory,
			IsTextOnly:    info.IsTextOnly,
			IsOptional:    info.IsOptional,
			ModuleName:    info.ModuleName,
			ExerciseName:  info.ExerciseName,
			State:         currentState.String(),
			PendingAction: state.GetPendingAction(),
			Error:         state.GetLastError(),
		}

		if updateAvailable, updateVersion, updateNotes := state.GetUpdateAvailable(); updateAvailable {
			resp.UpdateAvailable = true
			resp.UpdateVersion = updateVersion
			resp.UpdateCommand = "tdl update"
			resp.UpdateReleaseNotes = updateNotes
		}

		includeLogs, _ := request.GetArguments()["include_logs"].(bool)
		if includeLogs {
			resp.Logs = state.OutputBuffer().String()
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleRunSolution(state *LoopState) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentState := state.GetState()

		switch currentState {
		case StateFailed, StateSucceeded:
			// Valid states for running
		default:
			return mcp.NewToolResultError(stateError(
				"Cannot run solution", currentState, "failed' or 'succeeded", state,
			)), nil
		}

		return sendCommand(state, CmdRunSolution, "Solution run triggered", 30*time.Second)
	}
}

func handleNextExercise(state *LoopState) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentState := state.GetState()

		if currentState != StateSucceeded {
			return mcp.NewToolResultError(stateError(
				"Cannot advance to next exercise", currentState, "succeeded", state,
			)), nil
		}

		syncSolution, _ := request.GetArguments()["sync_solution"].(bool)

		cmdType := CmdNextExercise
		msg := "Advancing to next exercise"
		if syncSolution {
			cmdType = CmdSyncAndNextExercise
			msg = "Syncing with example solution and advancing to next exercise"
		}

		return sendCommand(state, cmdType, msg, 5*time.Minute)
	}
}

func handleResetExercise(state *LoopState) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentState := state.GetState()

		switch currentState {
		case StateFailed, StateSucceeded:
			// Valid states for reset
		default:
			return mcp.NewToolResultError(stateError(
				"Cannot reset exercise", currentState, "failed' or 'succeeded", state,
			)), nil
		}

		info := state.GetExerciseInfo()
		if info.IsTextOnly {
			return mcp.NewToolResultError("Cannot reset exercise: text-only exercises have no files to reset."), nil
		}

		return sendCommand(state, CmdResetExercise, "Exercise reset triggered", 2*time.Minute)
	}
}

const (
	tallyFormURL    = "https://api.tally.so/forms/68OLbP/respond"
	tallyAPIVersion = "2025-01-15"

	// Tally form field UUIDs.
	fieldTraining = "04efdc68-0896-4806-bd08-54e2f6c4da68"
	fieldExercise = "406a2251-70fe-4dda-9cb7-07bac19aac2e"
	fieldClient   = "9d05878a-de56-48b2-89d6-1d478d13adf3"
	fieldCLIVer   = "f7dd6c29-2611-4aa1-9d92-89cd459d8cb5"
	fieldHistory  = "26d96642-b5d2-4e92-b81a-98410e7cce81"
	fieldRating   = "580b56e3-985c-42cd-ab77-65c68a454a76"
	fieldFeedback = "67854b93-b30c-4836-8371-6f7954c135a2"
)

type tallyRequest struct {
	SessionUuid    string         `json:"sessionUuid"`
	RespondentUuid string         `json:"respondentUuid"`
	Responses      map[string]any `json:"responses"`
	Captchas       map[string]any `json:"captchas"`
	IsCompleted    bool           `json:"isCompleted"`
	Password       *string        `json:"password"`
}

func handleSendFeedback(state *LoopState) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		training, _ := args["training"].(string)
		exercise, _ := args["exercise"].(string)
		history, _ := args["history"].(string)
		rating, _ := args["rating"].(float64)
		feedback, _ := args["feedback"].(string)

		if training == "" || exercise == "" || feedback == "" {
			return mcp.NewToolResultError("training, exercise, and feedback are required"), nil
		}

		// Auto-filled fields.
		cliVersion := state.GetCLIVersion()
		mcpName, mcpVersion := state.GetMCPClientInfo()
		clientInfo := strings.TrimSpace(mcpName + " " + mcpVersion)

		payload := tallyRequest{
			SessionUuid:    uuid.NewString(),
			RespondentUuid: uuid.NewString(),
			Responses: map[string]any{
				fieldTraining: training,
				fieldExercise: exercise,
				fieldClient:   clientInfo,
				fieldCLIVer:   cliVersion,
				fieldHistory:  history,
				fieldRating:   int(rating),
				fieldFeedback: feedback,
			},
			Captchas:    map[string]any{},
			IsCompleted: true,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal payload: %v", err)), nil
		}

		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, tallyFormURL, bytes.NewReader(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("tally-version", tallyAPIVersion)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logrus.WithError(err).Warn("Feedback submission failed")
			return mcp.NewToolResultError(fmt.Sprintf("HTTP request failed: %v", err)), nil
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 300 {
			logrus.WithField("status", resp.StatusCode).Warn("Feedback submission returned non-success status")
			return mcp.NewToolResultError(fmt.Sprintf("Tally API returned status %d", resp.StatusCode)), nil
		}

		return mcp.NewToolResultText("Feedback submitted successfully"), nil
	}
}

// stateError builds an error message for state-gated tool calls, appending
// the pending action (e.g. "Merge conflict decision needed. Go to CLI.") when one exists.
func stateError(action string, current ExerciseState, expected string, state *LoopState) string {
	msg := fmt.Sprintf("%s: current state is '%s'. Must be '%s'.", action, current, expected)
	if pa := state.GetPendingAction(); pa != "" {
		msg += " Pending action: " + pa
	}
	return msg
}

func sendCommand(state *LoopState, cmdType CommandType, successMsg string, responseTimeout time.Duration) (*mcp.CallToolResult, error) {
	resultCh := make(chan MCPResult, 1)

	cmd := MCPCommand{
		Type:     cmdType,
		ResultCh: resultCh,
	}

	// Send command with timeout — the loop must pick it up within 10 seconds
	select {
	case state.commandCh <- cmd:
		// Command sent, wait for response
	case <-time.After(10 * time.Second):
		return mcp.NewToolResultError("Timed out waiting for the training loop to accept the command. The loop may be busy."), nil
	}

	// Wait for the loop to respond
	select {
	case result := <-resultCh:
		if result.Error != "" {
			return mcp.NewToolResultError(result.Error), nil
		}
		msg := successMsg
		if result.Message != "" {
			msg = result.Message
		}
		// Piggyback the one-shot update notice on the first successful tool
		// call after detection. Gives agent clients a chance to relay it to
		// the user since mcp-go has no server-push channel.
		if state.ShouldShowUpdateNoticeMCP() {
			if updateAvailable, updateVersion, _ := state.GetUpdateAvailable(); updateAvailable {
				msg += fmt.Sprintf(
					"\n\nNote: a new CLI version (%s) is available. Run `tdl update` in your terminal.",
					updateVersion,
				)
				state.MarkUpdateNoticeShownMCP()
			}
		}
		return mcp.NewToolResultText(msg), nil
	case <-time.After(responseTimeout):
		msg := "Timed out waiting for the training loop to respond."
		if pa := state.GetPendingAction(); pa != "" {
			msg += " Pending action: " + pa
		}
		return mcp.NewToolResultError(msg), nil
	}
}
