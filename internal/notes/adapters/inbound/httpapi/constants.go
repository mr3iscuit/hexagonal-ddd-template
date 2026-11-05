package httpapi

const (
	categoryBusiness  = "business"
	categoryTechnical = "technical"

	codeNoteNotFound                 = "NOTE_NOT_FOUND"
	codeNoteModifiedConcurrently     = "NOTE_MODIFIED_CONCURRENTLY"
	codeNoteLimitReached             = "NOTE_LIMIT_REACHED"
	codeNoteIDRequired               = "NOTE_ID_REQUIRED"
	codeNoteTitleRequired            = "NOTE_TITLE_REQUIRED"
	codeNoteTitleTooLong             = "NOTE_TITLE_TOO_LONG"
	codeNoteBodyTooLong              = "NOTE_BODY_TOO_LONG"
	codeNoteTitleHasControlCharacter = "NOTE_TITLE_HAS_CONTROL_CHARACTER"
	codeNoteBodyHasControlCharacter  = "NOTE_BODY_HAS_CONTROL_CHARACTER"
	codeNoteAlreadyArchived          = "NOTE_ALREADY_ARCHIVED"
	codeNoteNotArchived              = "NOTE_NOT_ARCHIVED"
	codeNoteArchived                 = "NOTE_ARCHIVED"
	codeNoteReminderNotInFuture      = "NOTE_REMINDER_NOT_IN_FUTURE"
	codeNoteReminderNotScheduled     = "NOTE_REMINDER_NOT_SCHEDULED"
	codeMalformedRequest             = "MALFORMED_REQUEST"
	codeValidationFailed             = "VALIDATION_FAILED"
	codeUnsupportedMediaType         = "UNSUPPORTED_MEDIA_TYPE"
	codeRequestTooLarge              = "REQUEST_TOO_LARGE"
	codeRouteNotFound                = "ROUTE_NOT_FOUND"
	codeMethodNotAllowed             = "METHOD_NOT_ALLOWED"
	codeRequestCancelled             = "REQUEST_CANCELLED"
	codeInternalError                = "INTERNAL_ERROR"

	internalErrorMessage    = "internal error"
	routeNotFoundMessage    = "route not found"
	methodNotAllowedMessage = "method not allowed"
	validationFailedMessage = "validation failed"
	requestTooLargeMessage  = "request body is too large"
	malformedBodyMessage    = "body is not valid JSON"
	wrongContentTypeMessage = "content type must be application/json"
	unexpectedFieldMessage  = "unexpected property"
	repeatedPropertyMessage = "a property appears more than once"
	missingPropertyFormat   = "expected required property %s to be present"
	wrongTypeFormat         = "%s has the wrong type"
	unexpectedBodyMessage   = "this request takes no body"
	unexpectedQueryMessage  = "unexpected query parameter"
	malformedQueryMessage   = "query string is not valid"
	invalidArchivedMessage  = "archived must be true or false, once"
	requestCancelledMessage = "request was cancelled"

	logRequestHandled   = "request handled"
	logRequestFailed    = "request failed"
	logRequestCancelled = "request cancelled"
	logPanicRecovered   = "panic recovered"
	logWriteFailed      = "write response failed"
	logKeyRequestID     = "requestId"
	logKeyError         = "error"
	logKeyMethod        = "method"
	logKeyPath          = "path"
	logKeyStatus        = "status"
	logKeyPanic         = "panic"
	logKeyStack         = "stack"

	bodyLocation      = "body"
	queryLocation     = "query"
	locationSeparator = "."

	titleField = "title"
	bodyField  = "body"
	jsonNull   = "null"

	charsetParameter = "charset"
	utf8Charset      = "utf-8"

	allowedMethodSeparator = ", "

	noteIDParameter   = "id"
	archivedParameter = "archived"
	archivedTrue      = "true"
	archivedFalse     = "false"

	// Every character of the largest valid note can need twelve bytes of JSON,
	// so this limit fits a full title and body with room for the field names.
	maximumRequestBodyBytes = 1 << 17
	maximumRequestIDLength  = 128

	// A JSON escape such as \ud83d is a two character prefix and four hex digits
	// that make one UTF-16 code unit.
	unicodeEscapePrefixLength = 2
	unicodeEscapeLength       = 6
	hexadecimalBase           = 16
	codeUnitBits              = 16
)
