package fetcherrors

import (
	"fmt"
)

// PkgMetaError indicates an error fetching, verifying and using a Pkg meta
// file.
type PkgMetaError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error).
func (e PkgMetaError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}

// PkgPrecheckError indicates an error prechecking a Pkg; this involves all
// operations done on Pkg meta info before a fetch is attempted. It includes
// checking the structure of the JSON and some of its values for consistency.
type PkgPrecheckError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error).
func (e PkgPrecheckError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}

// PkgSourceFetchAuthError indicates either an authentication or
// authorization error when fetching a Pkg from sources. It is expected to be
// returned only if all sources fail to fetch not for any single of multiple
// sources. An authentication error from an HTTP fetch is indicated by a 401
// from a source server; an authorization error from an HTTP fetch is indicated
// by a 403.
type PkgSourceFetchAuthError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error)
func (e PkgSourceFetchAuthError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}

// PkgSourceFetchError indicates a generic (non-auth) error fetching a part
// from provided sources. This is a more general error than
// PkgSourceFetchAuthError. Like that more specific error, this should only
// be produced if the fetcher failed to fetch a part from all sources, not
// any one of many possible sources.
type PkgSourceFetchError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error)
func (e PkgSourceFetchError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}

// PkgSourceError indicates a generic error handling Pkg sources not specific
// to fetching or verification. This may include errors writing Pkg Metadata
// or Parts to disk or otherwise processing them.
type PkgSourceError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error)
func (e PkgSourceError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}

// PkgSignatureVerificationError indicates a failure to verify a Pkg Part's
// signature(s). If more than one cryptographic signature is provided for
// verification, all must match one configured key or verification will fail.
type PkgSignatureVerificationError struct {
	Msg           string
	InternalError error
}

// Error provides a loggable error message including the message of an
// internal error (one enclosed in this error)
func (e PkgSignatureVerificationError) Error() string {
	return fmt.Sprintf("%v. InternalError: %v", e.Msg, e.InternalError)
}
