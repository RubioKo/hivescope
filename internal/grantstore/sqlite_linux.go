// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

/*
#cgo LDFLAGS: -lsqlite3
#include <sqlite3.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

type index struct{ db *C.sqlite3 }

func sqliteError(code C.int) error {
	return fmt.Errorf("grant index: %s (%d)", C.GoString(C.sqlite3_errstr(code)), int(code))
}

func openIndex(path string, create bool) (*index, error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var db *C.sqlite3
	// Creation is performed separately with O_EXCL and private permissions.
	rc := C.sqlite3_open_v2(name, &db, C.SQLITE_OPEN_READWRITE|C.SQLITE_OPEN_NOFOLLOW|C.SQLITE_OPEN_PRIVATECACHE, nil)
	if rc != C.SQLITE_OK {
		C.sqlite3_close(db)
		return nil, sqliteError(rc)
	}
	C.sqlite3_limit(db, C.SQLITE_LIMIT_LENGTH, 16384)
	C.sqlite3_limit(db, C.SQLITE_LIMIT_SQL_LENGTH, 16384)
	idx := &index{db}
	if err := idx.exec("PRAGMA synchronous=EXTRA; PRAGMA trusted_schema=OFF"); err != nil {
		idx.close()
		return nil, err
	}
	mode := "PRAGMA journal_mode"
	if create {
		mode = "PRAGMA journal_mode=DELETE"
	}
	for _, setting := range []struct{ query, want string }{
		{"PRAGMA synchronous", "3"},
		{"PRAGMA trusted_schema", "0"},
		{mode, "delete"},
		{"PRAGMA max_page_count=32", "32"},
	} {
		if err := idx.expect(setting.query, setting.want); err != nil {
			idx.close()
			return nil, err
		}
	}
	return idx, nil
}

func (i *index) close() error {
	if rc := C.sqlite3_close(i.db); rc != C.SQLITE_OK {
		return sqliteError(rc)
	}
	return nil
}

func (i *index) exec(query string) error {
	q := C.CString(query)
	defer C.free(unsafe.Pointer(q))
	if rc := C.sqlite3_exec(i.db, q, nil, nil, nil); rc != C.SQLITE_OK {
		return sqliteError(rc)
	}
	return nil
}

// scalar accepts exactly one value. Both TEXT pragma results and BLOB
// records have an exact byte representation. An explicit zero-length BLOB
// denotes a newly provisioned, empty grant; a missing row is an error.
func (i *index) scalar(query string) ([]byte, error) {
	q := C.CString(query)
	defer C.free(unsafe.Pointer(q))
	var stmt *C.sqlite3_stmt
	if rc := C.sqlite3_prepare_v2(i.db, q, -1, &stmt, nil); rc != C.SQLITE_OK {
		return nil, sqliteError(rc)
	}
	defer C.sqlite3_finalize(stmt)
	if rc := C.sqlite3_step(stmt); rc != C.SQLITE_ROW {
		return nil, fmt.Errorf("missing or unreadable grant index row: %w", sqliteError(rc))
	}
	if C.sqlite3_column_type(stmt, 0) == C.SQLITE_NULL {
		return nil, errors.New("unexpected NULL in grant index")
	}
	p := C.sqlite3_column_blob(stmt, 0)
	n := C.sqlite3_column_bytes(stmt, 0)
	if n > 512 || (n != 0 && p == nil) {
		return nil, errors.New("unexpected grant index value")
	}
	value := C.GoBytes(p, n)
	if rc := C.sqlite3_step(stmt); rc != C.SQLITE_DONE {
		return nil, fmt.Errorf("unexpected additional grant index row: %w", sqliteError(rc))
	}
	return value, nil
}

func (i *index) expect(query, want string) error {
	b, err := i.scalar(query)
	if err != nil {
		return err
	}
	if string(b) != want {
		return errors.New("unexpected SQLite grant index setting or inconsistent state")
	}
	return nil
}
