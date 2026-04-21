package trainings

// Accessors that route update-state reads/writes to LoopState when MCP is
// enabled, and to the local fields on Handlers otherwise. Callers (the
// background goroutine and the prompt site) stay agnostic to MCP mode.

func (h *Handlers) setUpdateAvailable(version, releaseNotes string) {
	if h.loopState != nil {
		h.loopState.SetUpdateAvailable(version, releaseNotes)
		return
	}
	h.updateMu.Lock()
	defer h.updateMu.Unlock()
	if h.updateVersion != version {
		h.updateNoticeShownCLI = false
	}
	h.updateAvailable = true
	h.updateVersion = version
	h.updateReleaseNotes = releaseNotes
}

func (h *Handlers) getUpdateAvailable() (available bool, version, releaseNotes string) {
	if h.loopState != nil {
		return h.loopState.GetUpdateAvailable()
	}
	h.updateMu.Lock()
	defer h.updateMu.Unlock()
	return h.updateAvailable, h.updateVersion, h.updateReleaseNotes
}

func (h *Handlers) shouldShowUpdateNoticeCLI() bool {
	if h.loopState != nil {
		return h.loopState.ShouldShowUpdateNoticeCLI()
	}
	h.updateMu.Lock()
	defer h.updateMu.Unlock()
	return h.updateAvailable && !h.updateNoticeShownCLI
}

func (h *Handlers) markUpdateNoticeShownCLI() {
	if h.loopState != nil {
		h.loopState.MarkUpdateNoticeShownCLI()
		return
	}
	h.updateMu.Lock()
	defer h.updateMu.Unlock()
	h.updateNoticeShownCLI = true
}
