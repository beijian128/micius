package iohook

import (
	"bytes"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

// Hook io Writer for logrus
type Hook struct {
	w    io.Writer
	fmt  logrus.Formatter
	pool sync.Pool
}

// New Hook
func New(w io.Writer, fmt logrus.Formatter) *Hook {
	result := new(Hook)
	result.w = w
	result.fmt = fmt
	result.pool.New = func() any {
		return new(bytes.Buffer)
	}
	return result
}

// Levels logrus.Hook interface
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire logrus.Hook interface
func (h *Hook) Fire(entry *logrus.Entry) error {
	buffer := h.pool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer h.pool.Put(buffer)
	entry.Buffer = buffer
	serialized, err := h.fmt.Format(entry)
	entry.Buffer = nil
	_, err = h.w.Write(serialized)
	return err
}
