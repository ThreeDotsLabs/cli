package mcp

import (
	"sync"
)

// ExerciseState represents the current state of the interactive run loop.
type ExerciseState int

const (
	StateIdle      ExerciseState = iota // Before first run
	StateRunning                        // runExercise is executing
	StateSucceeded                      // Last run passed, waiting at success prompt
	StateFailed                         // Last run failed, waiting at fail prompt
	StateAdvancing                      // nextExercise is in progress
)

func (s ExerciseState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateSucceeded:
		return "succeeded"
	case StateFailed:
		return "failed"
	case StateAdvancing:
		return "advancing"
	default:
		return "unknown"
	}
}

// ExerciseInfo contains the current exercise metadata exposed via MCP.
// This is a separate struct from config.ExerciseConfig to decouple the MCP API from internal config.
type ExerciseInfo struct {
	ExerciseID   string `json:"exercise_id"`
	Directory    string `json:"directory"`
	IsTextOnly   bool   `json:"is_text_only"`
	IsOptional   bool   `json:"is_optional"`
	ModuleName   string `json:"module_name"`
	ExerciseName string `json:"exercise_name"`
}

// CommandType represents a semantic command from MCP to the loop.
type CommandType int

const (
	CmdRunSolution CommandType = iota
	CmdNextExercise
	CmdSyncAndNextExercise
	CmdResetExercise
)

func (c CommandType) String() string {
	switch c {
	case CmdRunSolution:
		return "run_solution"
	case CmdNextExercise:
		return "next_exercise"
	case CmdSyncAndNextExercise:
		return "sync_and_next_exercise"
	case CmdResetExercise:
		return "reset_exercise"
	default:
		return "unknown"
	}
}

// MCPCommand is sent from MCP tool handlers to the loop via the command channel.
type MCPCommand struct {
	Type     CommandType
	ResultCh chan<- MCPResult
}

// MCPResult is sent back from the loop to the MCP tool handler.
type MCPResult struct {
	Success bool
	Message string
	Error   string
}

// LoopState is the shared state between the MCP server and the interactive run loop.
// All methods are thread-safe.
type LoopState struct {
	mu           sync.RWMutex
	state        ExerciseState
	exerciseInfo ExerciseInfo
	outputBuf    *OutputBuffer
	commandCh    chan MCPCommand

	cliVersion string

	mcpClientName    string
	mcpClientVersion string

	pendingAction string
	lastError     string

	transitionContent string // human-readable context from last exercise transition (e.g. diff)
}

func NewLoopState() *LoopState {
	return &LoopState{
		outputBuf: NewOutputBuffer(DefaultMaxOutputSize),
		commandCh: make(chan MCPCommand),
	}
}

func (s *LoopState) SetState(state ExerciseState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *LoopState) GetState() ExerciseState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *LoopState) SetExerciseInfo(info ExerciseInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exerciseInfo = info
}

func (s *LoopState) GetExerciseInfo() ExerciseInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exerciseInfo
}

func (s *LoopState) OutputBuffer() *OutputBuffer {
	return s.outputBuf
}

// CommandCh returns the channel for receiving MCP commands in the loop.
func (s *LoopState) CommandCh() <-chan MCPCommand {
	return s.commandCh
}

// SendCommand sends a command from MCP to the loop. Blocks until the loop processes it.
func (s *LoopState) SendCommand(cmd MCPCommand) {
	s.commandCh <- cmd
}

func (s *LoopState) SetCLIVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cliVersion = version
}

func (s *LoopState) GetCLIVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cliVersion
}

func (s *LoopState) SetMCPClientInfo(name, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpClientName = name
	s.mcpClientVersion = version
}

func (s *LoopState) GetMCPClientInfo() (name, version string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mcpClientName, s.mcpClientVersion
}

// SetPendingAction records that the CLI is blocked on a stdin prompt that
// the MCP client cannot resolve. Only use this for prompts with NO MCP
// channel — never for waitForAction() prompts which already multiplex
// stdin + MCP commands.
func (s *LoopState) SetPendingAction(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAction = msg
}

func (s *LoopState) GetPendingAction() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pendingAction
}

func (s *LoopState) ClearPendingAction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAction = ""
}

func (s *LoopState) SetLastError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = msg
}

func (s *LoopState) GetLastError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *LoopState) ClearLastError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = ""
}

func (s *LoopState) SetTransitionContent(content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitionContent = content
}

func (s *LoopState) GetTransitionContent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transitionContent
}

func (s *LoopState) ClearTransitionContent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitionContent = ""
}
