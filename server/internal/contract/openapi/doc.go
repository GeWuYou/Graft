// Package openapi owns non-runtime Go contract artifacts generated from the repository root OpenAPI spec.
//
// This boundary is intentionally isolated from plugin runtime wiring, handwritten HTTP DTO truth, and handler
// lifecycle ownership. Generated code here may be used for compile/test-only comparison during governance spikes, but
// it must not become implicit runtime truth without a separate approved topic.
package openapi
