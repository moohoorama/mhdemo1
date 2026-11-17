package mobile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Save times show in the phone's own zone: Go reads none on Android and would show UTC.
func TestSetZone(t *testing.T) {
	was := time.Local
	defer func() { time.Local = was }()
	time.Local = time.UTC
	setZone("Asia/Seoul")
	at := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC).Local()
	assert.Equal(t, 9, at.Hour(), "Seoul is UTC+9")
	setZone("Nowhere/Nothing")
	assert.Equal(t, "Asia/Seoul", time.Local.String(), "an unknown zone keeps the one there is")
}
