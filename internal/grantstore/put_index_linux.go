// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func uploadSlotName(slot uint64, kind byte) string {
	return fmt.Sprintf("item-%d-%d", slot, kind)
}

// Only local row numbers and the two fixed roles may name a slot. All
// operations also check the corresponding row and the private file metadata.
func parseUploadSlot(name string) (uint64, byte, bool) {
	number, role, ok := strings.Cut(strings.TrimPrefix(name, "item-"), "-")
	slot, err := strconv.ParseUint(number, 10, 8)
	if !ok || err != nil || slot == 0 || slot > uploadCeiling || (role != "1" && role != "2") {
		return 0, 0, false
	}
	kind := role[0] - '0'
	return slot, kind, name == uploadSlotName(slot, kind)
}

func (s *Store) reopenUploadSlots() error {
	version, err := s.uploadNumber("PRAGMA user_version")
	if err != nil {
		return err
	}
	if version == 1 {
		// Migrate only the exact shipped reservation schema. Reopen never
		// creates a missing ledger or discards its previous request IDs.
		if err := s.db.expect(fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE name='upload_v1' AND sql=CAST(X'%x' AS TEXT)", uploadTable), "1"); err != nil {
			return err
		}
		if err := s.db.exec("BEGIN IMMEDIATE; " + stagingSchema + " COMMIT"); err != nil {
			return err
		}
	}
	if err := s.db.expect("SELECT count(*) FROM pragma_table_info('upload_v1') WHERE name IN ('mask','cleanup') AND type='INTEGER' AND \"notnull\"=1 AND dflt_value='0'", "2"); err != nil {
		return err
	}
	if version <= 2 {
		if err := s.db.exec("BEGIN IMMEDIATE; " + publicationSchema + " COMMIT"); err != nil {
			return err
		}
	}
	if err := s.db.expect("SELECT count(*) FROM pragma_table_info('upload_v1') WHERE name='receipt' AND type='BLOB' AND \"notnull\"=1 AND hex(dflt_value)='582727'", "1"); err != nil {
		return err
	}
	// Check the terminal rows before any expiry cleanup. A contradictory
	// index must never turn a publication into reclaimable staged bytes.
	if err := s.db.expect(`SELECT count(*) FROM upload_v1 WHERE typeof(receipt)!='blob'
		OR NOT(length(receipt)=0 OR (length(receipt)=132 AND mask=3 AND expired=0 AND cleanup=0))`, "0"); err != nil {
		return err
	}
	if err := s.expireUploads(time.Now().Unix()); err != nil {
		return err
	}
	count, err := s.uploadNumber("SELECT count(*) FROM upload_v1 WHERE expired=0")
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		slot, err := s.uploadNumber(fmt.Sprintf("SELECT slot FROM upload_v1 WHERE expired=0 ORDER BY slot LIMIT 1 OFFSET %d", i))
		if err != nil || slot == 0 || slot > uploadCeiling {
			return errors.Join(errors.New("invalid upload slot"), err)
		}
		raw, err := s.db.scalar(fmt.Sprintf("SELECT manifest FROM upload_v1 WHERE slot=%d", slot))
		if err != nil || len(raw) != 276 {
			return errors.Join(errors.New("missing slot manifest"), err)
		}
		m := [276]byte(raw)
		published, _, _, err := s.publication(m, 0)
		if err != nil {
			return err
		}
		if published.Code != 0 {
			continue
		}
		if _, err := s.reconcileStagedPair(slot, m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) reconcileStagedPair(slot uint64, m [276]byte) (byte, error) {
	obj, cat, err := UploadItemSizes(m)
	if err != nil {
		return 0, err
	}
	catOK, err := s.reconcileSlot(uploadSlotName(slot, 1), cat, [32]byte(m[244:]))
	if err != nil {
		return 0, err
	}
	objOK, err := s.reconcileSlot(uploadSlotName(slot, 2), obj, [32]byte(m[172:204]))
	if err != nil {
		return 0, err
	}
	mask := byte(0)
	if catOK {
		mask |= 1
	}
	if objOK {
		mask |= 2
	}
	// A previous process may have linked or unlinked a slot without
	// syncing the directory. Persist that outcome before updating mask.
	if err := s.dir.Sync(); err != nil {
		return 0, err
	}
	return mask, s.db.exec(fmt.Sprintf("UPDATE upload_v1 SET mask=%d WHERE slot=%d AND mask!=%d AND length(receipt)=0", mask, slot, mask))
}

func (s *Store) reconcileSlot(name string, size uint64, digest [32]byte) (bool, error) {
	f, err := s.verifiedSlot(name, size, digest)
	if err != nil {
		return false, err
	}
	if f != nil {
		return true, errors.Join(f.Sync(), f.Close())
	}
	return false, s.unlinkSlot(name)
}

func (s *Store) unlinkSlot(name string) error {
	if _, err := s.ownedFile(name); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return unix.Unlinkat(int(s.dir.Fd()), name, 0)
}

// Persist terminal status and time before cleanup. A separate durable commit
// releases the charge only after each owned file is gone and its directory
// has been flushed. The same sequence resumes safely after interruption.
func (s *Store) expireUploads(now int64) error {
	due, err := s.uploadNumber(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE expired=0 AND length(receipt)=0 AND deadline<=%d", now))
	if err != nil {
		return err
	}
	if due != 0 {
		if err := s.db.exec(fmt.Sprintf(`BEGIN IMMEDIATE;
			UPDATE upload_v1 SET expired=1,cleanup=1 WHERE expired=0 AND length(receipt)=0 AND deadline<=%d;
			UPDATE enrollment SET last_utc=%d WHERE id=1; COMMIT`, now, now)); err != nil {
			return err
		}
		s.lastUTC = now
	}
	remaining, err := s.uploadNumber("SELECT count(*) FROM upload_v1 WHERE cleanup=1")
	if err != nil || remaining > uploadCeiling {
		return errors.Join(errors.New("invalid pending cleanup count"), err)
	}
	for i := uint64(0); i < remaining; i++ {
		slot, err := s.uploadNumber("SELECT slot FROM upload_v1 WHERE expired=1 AND cleanup=1 ORDER BY slot LIMIT 1")
		if err != nil || slot == 0 || slot > uploadCeiling {
			return errors.Join(errors.New("invalid expired slot"), err)
		}
		raw, err := s.db.scalar(fmt.Sprintf("SELECT id FROM upload_v1 WHERE slot=%d", slot))
		if err != nil || len(raw) != 16 {
			return errors.Join(errors.New("missing expired request identity"), err)
		}
		if w := s.writers[[16]byte(raw)]; w != nil {
			if err := w.abortLocked(); err != nil {
				return err
			}
		}
		for kind := byte(1); kind <= 2; kind++ {
			if err := s.unlinkSlot(uploadSlotName(slot, kind)); err != nil {
				return err
			}
		}
		if err := s.dir.Sync(); err != nil {
			return err
		}
		if err := s.db.exec(fmt.Sprintf("UPDATE upload_v1 SET mask=0,cleanup=0 WHERE slot=%d", slot)); err != nil {
			return err
		}
	}
	return nil
}
