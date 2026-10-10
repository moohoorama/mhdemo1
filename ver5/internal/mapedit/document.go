package mapedit

import (
	"encoding/json"
	"srpg/internal/content"
)

const Version = content.MapVersion
const MaxSize = content.MapMaxSize

type Document = content.MapDocument
type Rect = content.MapRect

func Import(st content.Stage) *Document { return content.ImportMap(st) }
func FloorHalf(x int) int               { return content.FloorHalf(x) }
func Key(x, y int) string               { return content.Key(x, y) }
func Clone[T any](v T) T                { b, _ := json.Marshal(v); var out T; _ = json.Unmarshal(b, &out); return out }

type History struct{ UndoStack, RedoStack [][]byte }

func (h *History) Push(d *Document) {
	b, _ := json.Marshal(d)
	h.UndoStack = append(h.UndoStack, b)
	if len(h.UndoStack) > 20 {
		h.UndoStack = h.UndoStack[1:]
	}
	h.RedoStack = nil
}
func (h *History) Undo(d **Document) bool { return h.swap(d, &h.UndoStack, &h.RedoStack) }
func (h *History) Redo(d **Document) bool { return h.swap(d, &h.RedoStack, &h.UndoStack) }
func (h *History) swap(d **Document, from, to *[][]byte) bool {
	if len(*from) == 0 {
		return false
	}
	b, _ := json.Marshal(*d)
	*to = append(*to, b)
	n := len(*from) - 1
	var doc Document
	if json.Unmarshal((*from)[n], &doc) != nil {
		return false
	}
	*from = (*from)[:n]
	*d = &doc
	return true
}
