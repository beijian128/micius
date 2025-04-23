package protoreq

import (
	"fmt"
)

// Error interface
func (m *ErrCode) Error() string {
	if m.Msg == "" {
		return fmt.Sprintf("code=%d", m.Code)
	}
	return fmt.Sprintf("code=%d msg=%s", m.Code, m.Msg)
}
