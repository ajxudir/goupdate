package cmd

// resetUpdateFlagsToDefaults is a test helper that resets all update flags to their default values.
//
// This function ensures test isolation by resetting all update command flags to their initial state.
func resetUpdateFlagsToDefaults() {
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateRuleFlag = "all"
	updateNameFlag = ""
	updateGroupFlag = ""
	updateConfigFlag = ""
	updateDirFlag = "."
	updateFileFlag = ""
	updateMajorFlag = false
	updateMinorFlag = false
	updatePatchFlag = false
	updateIncrementalFlag = false
	updateDryRunFlag = false
	updateSkipLockRun = false
	updateYesFlag = false
	updateNoTimeoutFlag = false
	updateContinueOnFail = false
	updateSkipPreflight = false
	updateOutputFlag = ""
	updateSkipSystemTests = false
	updateSystemTestModeFlag = ""
}
