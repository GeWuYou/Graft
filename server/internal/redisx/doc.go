// Package redisx opens the Redis client used by the core runtime.
//
// Redis is initialized before plugin boot so later cache, session, rate-limit,
// and scheduler primitives can share one visible infrastructure handle.
package redisx
