// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package logknob provides a helpful wrapper that allows enabling logging
// based on either an envknob or other methods of enablement.
package logknob

import (
	"sync/atomic"

	"tailscale.com/envknob"
	"tailscale.com/types/logger"
)

// TODO(andrew-d): should we have a package-global registry of logknobs? It
// would allow us to update from a netmap in a central location, which might be
// reason enough to do it...

// LogKnob allows configuring verbose logging, with multiple ways to enable. It
// supports enabling logging via envknob, via atomic boolean (for use in e.g.
// c2n log level changes), and via capabilities from a NetMap (so users can
// enable logging via the ACL JSON).
type LogKnob struct {
	capName string
	cap     atomic.Bool
	env     func() bool
	manual  atomic.Bool
}

// NewLogKnob creates a new LogKnob, with the provided environment variable
// name and/or NetMap capability.
func NewLogKnob(env, cap string) *LogKnob {
	if env == "" && cap == "" {
		panic("must provide either an environment variable or capability")
	}

	l := &LogKnob{
		capName: cap,
	}
	if env != "" {
		l.env = envknob.RegisterBool(env)
	} else {
		l.env = func() bool { return false }
	}
	return l
}

// Set will cause logs to be printed when called with Set(true). When called
// with Set(false), logs will not be printed due to an earlier call of
// Set(true), but may be printed due to either the envknob and/or capability of
// this LogKnob.
func (l *LogKnob) Set(v bool) {
	l.manual.Store(v)
}

// NetMap is an interface for the parts of netmap.NetworkMap that we care
// about; we use this rather than a concrete type to avoid a circular
// dependency.
type NetMap interface {
	SelfCapabilities() []string
}

// UpdateFromNetMap will enable logging if the SelfNode in the provided NetMap
// contains the capability provided for this LogKnob.
func (l *LogKnob) UpdateFromNetMap(nm NetMap) {
	if l.capName == "" {
		return
	}

	for _, c := range nm.SelfCapabilities() {
		if c == l.capName {
			l.cap.Store(true)
			return
		}
	}

	l.cap.Store(false)
}

// Do will call log with the provided format and arguments if any of the
// configured methods for enabling logging are true.
func (l *LogKnob) Do(log logger.Logf, format string, args ...any) {
	if l.shouldLog() {
		log(format, args...)
	}
}

func (l *LogKnob) shouldLog() bool {
	return l.manual.Load() || l.env() || l.cap.Load()
}
