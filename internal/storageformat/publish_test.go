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
		t.Fatalf("child unexpectedly resumed: %v", s.saveObject(bytes.NewReader(object), filepath.Join(childDir, destination), true, "", pause, syncPublishDir))
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
	const childFlag = "HIVESCOPE_TEST_UNCERTAIN_RESTORE"
	names := []string{"unchanged.bin", "edited.bin"}
	if dir := os.Getenv(childFlag); dir != "" {
		if !filepath.IsAbs(dir) {
			t.Fatal("expected private test directory")
		}
		snapshot, object := publicationVector(t)
		flushErr := errors.New("injected directory sync failure")
		for _, name := range names {
			err := snapshot.saveObject(bytes.NewReader(object), filepath.Join(dir, name), true,
				filepath.Join(dir, name+".state"), publishStaged, func(*os.File) error { return flushErr })
			if !errors.Is(err, ErrPublicationUncertain) || !errors.Is(err, flushErr) {
				t.Fatalf("unexpected post-rename result: %v", err)
			}
		}
		return
	}
	dir := t.TempDir()
	for _, name := range names {
		if err := os.Mkdir(filepath.Join(dir, name+".state"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("keep\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, exe, "-test.run=^TestFailureAfterReplaceIsUncertain$")
	child.Dir = dir
	child.Env = append(child.Environ(), childFlag+"="+dir)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("saving process: %v\n%s", err, out)
	}
	// Both saves were uncertain when the child exited. Only the second
	// destination is edited before the first reconciliation of either save.
	if err := os.WriteFile(filepath.Join(dir, "edited.bin"), []byte("Hive!"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, object := publicationVector(t)
	for _, name := range names {
		dest := filepath.Join(dir, name)
		want := "Hive\n"
		var wantErr error
		if name == "edited.bin" {
			want, wantErr = "Hive!", ErrDestinationDiffers
		}
		err := snapshot.ReopenPublication(bytes.NewReader(object), dest, filepath.Join(dir, name+".state"))
		if !errors.Is(err, wantErr) {
			t.Fatalf("%s: reopen returned %v, want %v", name, err, wantErr)
		}
		got, err := os.ReadFile(dest)
		if err != nil || string(got) != want {
			t.Fatalf("%s after reopen = %q, error = %v", name, got, err)
		}
	}
}

func TestReopenRequiresBothFlushes(t *testing.T) {
	snapshot, object := publicationVector(t)
	destination, state := filepath.Join(t.TempDir(), "chosen.bin"), t.TempDir()
	if err := snapshot.SaveRecoverableObject(bytes.NewReader(object), destination, false, state); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected flush failure")
	for _, failAt := range []string{"file", "directory", ""} {
		var order []string
		flush := func(label string) func(*os.File) error {
			return func(f *os.File) error {
				order = append(order, label)
				if label == failAt {
					return injected
				}
				return f.Sync()
			}
		}
		err := snapshot.reopenPublication(bytes.NewReader(object), destination, state, flush("file"), flush("directory"))
		wantOrder := "file directory"
		if failAt == "file" {
			wantOrder = "file"
		}
		if strings.Join(order, " ") != wantOrder {
			t.Fatalf("failure at %q: flush order %v", failAt, order)
		}
		if failAt == "" && err != nil || failAt != "" && (!errors.Is(err, ErrPublicationUncertain) || !errors.Is(err, injected)) {
			t.Fatalf("failure at %q: unexpected reconciliation result: %v", failAt, err)
		}
	}
}
