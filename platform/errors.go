package platform

import "errors"

// Errors returned by the platform package.
var (
	// Configuration errors
	ErrNoDatabaseURL = errors.New("platform: DATABASE_URL required in platform mode")
	ErrNoServiceID   = errors.New("platform: SERVICE_ID required")

	// Access errors
	ErrAccessDenied = errors.New("platform: access denied")
	ErrUserNotFound = errors.New("platform: user not found")
	ErrUserDisabled = errors.New("platform: user disabled")

	// Quota errors
	ErrQuotaExceeded = errors.New("platform: quota exceeded")
	ErrNoQuotaFound  = errors.New("platform: no quota found for user")

	// Admin errors
	ErrNotAdmin = errors.New("platform: user is not an admin")
)
