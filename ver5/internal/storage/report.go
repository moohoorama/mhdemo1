package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	syncatomic "sync/atomic"
	"time"

	"srpg/internal/core"
)

// MaxReports is how many report files the reports directory keeps; later ones are dropped until it is emptied
// (by an upload, say).
const MaxReports = 50

var (
	reported sync.Map         // the warning sets this run already wrote
	sequence syncatomic.Int64 // tells apart reports written in the same millisecond
)

// Report is one problem the game played on through, as written under reports/ in the save directory.
type Report struct {
	Time     string     `json:"time"`
	Kind     string     `json:"kind"` // "restore": a save or the move preview held rule mismatches (core.RestoreWarned)
	Warnings []string   `json:"warnings"`
	State    core.State `json:"state"`
}

// ReportRestore keeps the rule mismatches a release build let a restore pass with, and the state they were found
// in. Within a run each set is written once (the move preview restores the same state again and again).
func (s Store) ReportRestore(warnings []string, state core.State) error {
	if _, dup := reported.LoadOrStore(strings.Join(warnings, "\n"), true); dup {
		return nil
	}
	dir := filepath.Join(s.Dir, "reports")
	if files, _ := os.ReadDir(dir); len(files) >= MaxReports {
		return nil
	}
	now := time.Now()
	b, err := json.Marshal(Report{Time: now.UTC().Format(time.RFC3339), Kind: "restore", Warnings: warnings, State: state})
	if err != nil {
		return err
	}
	return atomic(filepath.Join(dir, fmt.Sprintf("restore-%s-%d.json", now.Format("20060102-150405.000"), sequence.Add(1))), b)
}
