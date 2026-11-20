// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows || (linux && !android)

package storageformat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Public five-byte hive.txt vector from file-catalog.md. The conformance
// suite covers obtaining this snapshot through the signed catalog.
func publicationVector(t *testing.T) (*Snapshot, []byte) {
	t.Helper()
	object, err := hex.DecodeString("bac6d517956355ed4498a00c916ef93307ce8c69f8")
	if err != nil {
		t.Fatal(err)
	}
	return &Snapshot{
		folderID:     [16]byte(bytes.Repeat([]byte{0xa1}, 16)),
		versionID:    [16]byte(bytes.Repeat([]byte{0x71}, 16)),
		versionKey:   [32]byte(bytes.Repeat([]byte{0x81}, 32)),
		objectDigest: sha256.Sum256(object),
		size:         5,
	}, object
}

func TestProcessInterruptionBeforeReplace(t *testing.T) {
	const childFlag = "HIVESCOPE_TEST_PAUSE_BEFORE_RENAME"
	const destination = "selected.txt"
	if childDir := os.Getenv(childFlag); childDir != "" {
		if !filepath.IsAbs(childDir) {
			t.Fatal("expected a private test directory")
		}
		s, object := publicationVector(t)
		pause := func(_ *os.File, stage, _ string, _ bool) error {
			// At this point the real verification, file sync and close have
			// succeeded. Wait at the entry to the one native rename operation.
			fmt.Fprintln(os.Stdout, "ready", stage)
			var resume [1]byte
			if _, err := io.ReadFull(os.Stdin, resume[:]); err != nil {
				return err
			}
			return errors.New("parent unexpectedly released the pre-rename barrier")
		}
		t.Fatalf("child unexpectedly resumed: %v", s.saveObject(bytes.NewReader(object), filepath.Join(childDir, destination), true, pause, syncPublishDir))
	}

	dir := t.TempDir()
	target := filepath.Join(dir, destination)
	previous := []byte("keep\n")
	if err := os.WriteFile(target, previous, 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestProcessInterruptionBeforeReplace$")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), childFlag+"="+dir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	// The process cannot move past the barrier until a byte is written to
	// stdin. Kill it instead, without running any deferred cleanup.
	line, readErr := bufio.NewReader(out).ReadString('\n')
	stage, ready := strings.CutPrefix(strings.TrimSpace(line), "ready ")
	if readErr != nil || !ready || !filepath.IsLocal(stage) || stage == destination {
		t.Fatalf("child failed to reach pre-rename barrier: %q (%v)", line, readErr)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	var terminated *exec.ExitError
	if err := cmd.Wait(); !errors.As(err, &terminated) || ctx.Err() != nil {
		t.Fatalf("expected deliberate process termination: %v (%v), %s", err, ctx.Err(), stderr.String())
	}
	// Reopen by the selected name after the child has exited. Also establish
	// that it really reached a complete, verified, separately staged file.
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, previous) {
		t.Fatalf("previous destination after process kill = %q, error = %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(dir, stage))
	if err != nil || string(got) != "Hive\n" {
		t.Fatalf("verified temporary after process kill = %q, error = %v", got, err)
	}
}

func TestFailureAfterReplaceIsUncertain(t *testing.T) {
	snapshot, object := publicationVector(t)
	destination := filepath.Join(t.TempDir(), "chosen.bin")
	if err := os.WriteFile(destination, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	flushErr := errors.New("injected directory sync failure")
	err := snapshot.saveObject(bytes.NewReader(object), destination, true, publishStaged,
		func(*os.File) error { return flushErr })
	if !errors.Is(err, ErrPublicationUncertain) || !errors.Is(err, flushErr) {
		t.Fatalf("unexpected post-rename result: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "Hive\n" {
		t.Fatalf("visible destination after failed directory sync = %q, error = %v", got, err)
	}
}
