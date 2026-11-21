// SPDX-License-Identifier: AGPL-3.0-only

// Package grantstore persists the provider's highest signed grant for one
// locally provisioned agreement. Its caller verifies signatures and scope.
package grantstore

import "errors"

var ErrUnsupported = errors.New("durable grant storage requires Linux with SQLite and cgo")

// Create exclusively provisions a new private directory under an existing
// trusted parent. binding identifies the enrolled store and agreement.
func Create(dir string, binding [32]byte) (*Store, error) { return open(dir, binding, true) }

// Open reopens an existing provisioned store. Missing or inconsistent state
// requires reconciliation. It must never be initialized as a new enrollment.
func Open(dir string, binding [32]byte) (*Store, error) { return open(dir, binding, false) }
