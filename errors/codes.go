package errors

// Standard error codes for Cloistr services.
// Services should use these codes where applicable for consistency.
// Services may define additional codes for service-specific errors.

// Authentication and authorization errors (AUTH_*)
const (
	CodeAuthRequired       = "AUTH_REQUIRED"
	CodeAuthInvalid        = "AUTH_INVALID"
	CodeAuthExpired        = "AUTH_EXPIRED"
	CodeAccessDenied       = "ACCESS_DENIED"
	CodeNotAdmin           = "NOT_ADMIN"
	CodeUserDisabled       = "USER_DISABLED"
	CodeUserNotFound       = "USER_NOT_FOUND"
	CodeSignatureInvalid   = "SIGNATURE_INVALID"
	CodeSignatureExpired   = "SIGNATURE_EXPIRED"
	CodeNIP46Error         = "NIP46_ERROR"
	CodeNIP46Timeout       = "NIP46_TIMEOUT"
	CodeNIP46Rejected      = "NIP46_REJECTED"
)

// Quota and rate limiting errors (QUOTA_*, RATE_*)
const (
	CodeQuotaExceeded      = "QUOTA_EXCEEDED"
	CodeStorageFull        = "STORAGE_FULL"
	CodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	CodeRequestTooLarge    = "REQUEST_TOO_LARGE"
)

// Resource errors (RESOURCE_*)
const (
	CodeResourceNotFound   = "RESOURCE_NOT_FOUND"
	CodeResourceExists     = "RESOURCE_EXISTS"
	CodeResourceConflict   = "RESOURCE_CONFLICT"
	CodeResourceLocked     = "RESOURCE_LOCKED"
)

// Storage errors (STORAGE_*)
const (
	CodeStorageTimeout     = "STORAGE_TIMEOUT"
	CodeStorageError       = "STORAGE_ERROR"
	CodeStorageUnavailable = "STORAGE_UNAVAILABLE"
	CodeUploadFailed       = "UPLOAD_FAILED"
	CodeDownloadFailed     = "DOWNLOAD_FAILED"
)

// Validation errors (VALIDATION_*)
const (
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeInvalidInput       = "INVALID_INPUT"
	CodeInvalidFormat      = "INVALID_FORMAT"
	CodeMissingField       = "MISSING_FIELD"
	CodeInvalidPubkey      = "INVALID_PUBKEY"
)

// Relay errors (RELAY_*)
const (
	CodeRelayError         = "RELAY_ERROR"
	CodeRelayTimeout       = "RELAY_TIMEOUT"
	CodeRelayUnavailable   = "RELAY_UNAVAILABLE"
	CodeEventRejected      = "EVENT_REJECTED"
)

// Service errors (SERVICE_*)
const (
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeServiceTimeout     = "SERVICE_TIMEOUT"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeNotImplemented     = "NOT_IMPLEMENTED"
)

// Pre-built common errors for convenience.
var (
	// ErrAuthRequired is returned when authentication is required but not provided.
	ErrAuthRequired = Unauthorized(CodeAuthRequired, "authentication required")

	// ErrAuthInvalid is returned when authentication credentials are invalid.
	ErrAuthInvalid = Unauthorized(CodeAuthInvalid, "invalid authentication")

	// ErrAccessDenied is returned when the user lacks permission for the operation.
	ErrAccessDenied = Forbidden(CodeAccessDenied, "access denied")

	// ErrNotAdmin is returned when admin privileges are required.
	ErrNotAdmin = Forbidden(CodeNotAdmin, "admin privileges required")

	// ErrQuotaExceeded is returned when a user exceeds their quota.
	ErrQuotaExceeded = InsufficientStorage(CodeQuotaExceeded, "quota exceeded")

	// ErrResourceNotFound is returned when the requested resource doesn't exist.
	ErrResourceNotFound = NotFound(CodeResourceNotFound, "resource not found")

	// ErrInternalError is returned for unexpected server errors.
	ErrInternalError = InternalError(CodeInternalError, "internal server error")
)
