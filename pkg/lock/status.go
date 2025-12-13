package lock

const (
	InstallStatusLockFound      = "LockFound"
	InstallStatusNotConfigured  = "NotConfigured"
	InstallStatusNotInLock      = "NotInLock"
	InstallStatusLockMissing    = "LockMissing"
	InstallStatusVersionMissing = "VersionMissing"
	// InstallStatusSelfPinned indicates the manifest is self-pinning (e.g., requirements.txt)
	// and the declared version is used as the installed version.
	InstallStatusSelfPinned = "SelfPinned"
	// InstallStatusFloating indicates the version is a floating constraint (5.*, >=8.0.0, etc.)
	// that cannot be updated automatically. Users must either remove the floating constraint
	// or handle updates manually.
	InstallStatusFloating = "Floating"
	// InstallStatusIgnored indicates the package is excluded from processing based on
	// configuration (ignore patterns or package_overrides.ignore = true).
	// The package is still reported for visibility, but no updates will be performed.
	InstallStatusIgnored = "Ignored"
)
