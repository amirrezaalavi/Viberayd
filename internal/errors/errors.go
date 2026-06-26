package errors

import "errors"

// Sentinel errors for the parser engine
var (
	ErrInvalidProtocol   = errors.New("unsupported or unknown proxy protocol")
	ErrInvalidEncoding   = errors.New("invalid base64 or URL encoding")
	ErrInvalidFormat     = errors.New("malformed URI format")
	ErrMissingField      = errors.New("required field is missing")
	ErrInvalidUUID       = errors.New("invalid UUID format")
	ErrInvalidPort       = errors.New("port out of valid range (1-65535)")
	ErrInvalidNetwork    = errors.New("unsupported network type")
	ErrInvalidSecurity   = errors.New("unsupported security type")
	ErrInvalidFlow       = errors.New("unsupported flow value")
	ErrInvalidPublicKey  = errors.New("invalid Reality public key format")
	ErrInvalidFingerprint = errors.New("unsupported TLS fingerprint")
)

// Sentinel errors for the testing engine
var (
	ErrTCPConnect      = errors.New("TCP connection failed")
	ErrTLSHandshake    = errors.New("TLS handshake failed")
	ErrProtocolTest    = errors.New("protocol-specific test failed")
	ErrProxyTest       = errors.New("proxy test through Xray failed")
	ErrXrayNotFound    = errors.New("xray binary not found or not executable")
	ErrXrayCrash       = errors.New("xray process crashed unexpectedly")
	ErrPortExhausted   = errors.New("no available ports in allocation range")
	ErrPortConflict    = errors.New("port already in use")
	ErrResourceExhaust = errors.New("system resource exhausted")
)

// Sentinel errors for output / orchestration
var (
	ErrOutputWrite   = errors.New("failed to write output file")
	ErrInvalidDepth  = errors.New("invalid test depth specified")
	ErrInvalidStyle  = errors.New("invalid output style specified")
	ErrInterrupted   = errors.New("operation interrupted by user")
)

// Category wraps an error with a classification label.
type Category string

const (
	CategoryParse    Category = "parse"
	CategoryNetwork  Category = "network"
	CategoryProtocol Category = "protocol"
	CategoryResource Category = "resource"
	CategoryRuntime  Category = "runtime"
	CategoryUnknown  Category = "unknown"
)

// CategorizedError pairs an error with its failure category.
type CategorizedError struct {
	Category Category
	Err      error
	ConfigID string // optional identifier for the config that failed
}

func (ce CategorizedError) Error() string {
	if ce.ConfigID != "" {
		return string(ce.Category) + "[" + ce.ConfigID + "]: " + ce.Err.Error()
	}
	return string(ce.Category) + ": " + ce.Err.Error()
}

func (ce CategorizedError) Unwrap() error {
	return ce.Err
}

// Categorize inspects an error and assigns a Category.
func Categorize(err error, configID string) CategorizedError {
	ce := CategorizedError{Err: err, ConfigID: configID}
	switch {
	case errors.Is(err, ErrInvalidProtocol),
		errors.Is(err, ErrInvalidEncoding),
		errors.Is(err, ErrInvalidFormat),
		errors.Is(err, ErrMissingField),
		errors.Is(err, ErrInvalidUUID),
		errors.Is(err, ErrInvalidPort),
		errors.Is(err, ErrInvalidNetwork),
		errors.Is(err, ErrInvalidSecurity),
		errors.Is(err, ErrInvalidFlow),
		errors.Is(err, ErrInvalidPublicKey),
		errors.Is(err, ErrInvalidFingerprint):
		ce.Category = CategoryParse
	case errors.Is(err, ErrTCPConnect),
		errors.Is(err, ErrPortExhausted),
		errors.Is(err, ErrPortConflict):
		ce.Category = CategoryNetwork
	case errors.Is(err, ErrTLSHandshake),
		errors.Is(err, ErrProtocolTest):
		ce.Category = CategoryProtocol
	case errors.Is(err, ErrResourceExhaust):
		ce.Category = CategoryResource
	case errors.Is(err, ErrXrayCrash),
		errors.Is(err, ErrProxyTest),
		errors.Is(err, ErrXrayNotFound):
		ce.Category = CategoryRuntime
	default:
		ce.Category = CategoryUnknown
	}
	return ce
}
