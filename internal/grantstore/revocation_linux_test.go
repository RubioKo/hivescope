// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The protocol tests authenticate the full signature on each retry. Here the
// records have synthetic opaque contents and real v1 state / generation fields.
func interruptedStore(t *testing.T) (string, [32]byte, *Store, []byte, []byte) {
	t.Helper()
	dir, binding, s := newStore(t)
	active, revoked := recordAt(7), recordAt(8)
	active[100], active[101] = 1, 1
	revoked[100], revoked[101] = 2, 1
	if err := s.Replace(nil, active); err != nil {
		t.Fatal(err)
	}
	if err := s.publishIntent(active, revoked); err != nil {
		t.Fatal(err)
	}
	return dir, binding, s, active, revoked
}

func TestResumeExactRevocationAcrossWriteBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name        string
		rowAdvanced bool
		stageLength int
		anchorNew   bool
	}{
		{"durable intent before SQLite", false, -1, false},
		{"SQLite before checkpoint create", true, -1, false},
		{"empty staged digest", true, 0, false},
		{"partial staged digest", true, 9, false},
		{"complete staged digest", true, sha256.Size, false},
		{"checkpoint before intent removal", true, -1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, binding, s, active, revoked := interruptedStore(t)
			if err := s.finishIntent(); err == nil {
				t.Fatal("retired the intent before the SQLite transition")
			}
			if tc.rowAdvanced {
				if err := s.advanceRow(revoked, time.Now().Unix()); err != nil {
					t.Fatal(err)
				}
			}
			if tc.stageLength >= 0 {
				digest := sha256.Sum256(revoked)
				if err := os.WriteFile(filepath.Join(dir, stageName), digest[:tc.stageLength], 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.anchorNew {
				if err := s.writeAnchor(revoked); err != nil {
					t.Fatal(err)
				}
			}
			for attempt := 0; attempt < 2; attempt++ {
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				var err error
				s, err = Open(dir, binding)
				if err != nil {
					t.Fatal(err)
				}
				opened := s
				t.Cleanup(func() { opened.Close() })
				if record, err := s.Record(); !errors.Is(err, ErrPendingRevocation) || record != nil {
					t.Fatal("ordinary grant became available before retry")
				}
				if err := s.Replace(active, recordAt(9)); !errors.Is(err, ErrPendingRevocation) {
					t.Fatal("advanced beyond the pending revocation")
				}
				other := bytes.Clone(revoked)
				other[150] ^= 1
				if err := s.CompleteRevocation(other); err == nil {
					t.Fatal("completed using a different signed record")
				}
				if got, err := s.PendingRevocation(); err != nil || !bytes.Equal(got, revoked) {
					t.Fatal("lost the exact pending revocation")
				}
			}
			if err := s.CompleteRevocation(revoked); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(dir, binding)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if got, err := reopened.Record(); err != nil || !bytes.Equal(got, revoked) {
				t.Fatalf("unexpected record after final reopen: %v", err)
			}
			if got, err := reopened.PendingRevocation(); err != nil || got != nil {
				t.Fatal("confirmed retry left an intent")
			}
		})
	}
}

func TestInvalidTransitionEvidenceCannotResume(t *testing.T) {
	for _, scenario := range []string{"old index with new checkpoint", "unknown checkpoint",
		"unrelated checkpoint staging", "symlink staging", "out-of-order staging", "truncated intent", "foreign store intent"} {
		t.Run(scenario, func(t *testing.T) {
			dir, binding, s, _, revoked := interruptedStore(t)
			var err error
			switch scenario {
			case "old index with new checkpoint":
				err = s.writeAnchor(revoked)
			case "unknown checkpoint":
				err = s.writeAnchor(recordAt(19))
			case "unrelated checkpoint staging", "symlink staging":
				if err := s.advanceRow(revoked, time.Now().Unix()); err != nil {
					t.Fatal(err)
				}
				if scenario == "symlink staging" {
					err = os.Symlink(filepath.Join(dir, anchorName), filepath.Join(dir, stageName))
				} else {
					digest := sha256.Sum256(revoked)
					digest[0] ^= 1
					err = os.WriteFile(filepath.Join(dir, stageName), digest[:1], 0600)
				}
			case "out-of-order staging":
				err = os.WriteFile(filepath.Join(dir, stageName), nil, 0600)
			case "truncated intent":
				err = os.Truncate(filepath.Join(dir, intentName), 32)
			case "foreign store intent":
				foreignDir, _, _, _, _ := interruptedStore(t)
				marker, err := os.ReadFile(filepath.Join(foreignDir, intentName))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, intentName), marker, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(dir, binding); err == nil {
				reopened.Close()
				t.Fatal("reopened a grant with inconsistent transition evidence")
			}
		})
	}
}
