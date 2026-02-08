package api

// JSONKeyError is the response key used for API error codes.
const JSONKeyError = jsonKeyError

// ErrorValueInvalidJSON indicates a malformed JSON payload.
const ErrorValueInvalidJSON = errorValueInvalidJSON

// ErrorValueMissingFields indicates required input fields are missing.
const ErrorValueMissingFields = errorValueMissingFields

// ErrorValueSaveFailed indicates a persistence failure.
const ErrorValueSaveFailed = errorValueSaveFailed

// ErrorValueNotAuthorized indicates the caller lacks required permissions.
const ErrorValueNotAuthorized = errorValueNotAuthorized

// ErrorValueInvalidOwner indicates an invalid site owner value.
const ErrorValueInvalidOwner = errorValueInvalidOwner

// ErrorValueInvalidWidgetSide indicates an unsupported widget placement side.
const ErrorValueInvalidWidgetSide = errorValueInvalidWidgetSide

// ErrorValueInvalidWidgetOffset indicates an unsupported widget offset value.
const ErrorValueInvalidWidgetOffset = errorValueInvalidWidgetOffset

// ErrorValueSiteExists indicates a site creation conflict.
const ErrorValueSiteExists = errorValueSiteExists

// AuthErrorForbidden indicates an authenticated request was forbidden.
const AuthErrorForbidden = authErrorForbidden
