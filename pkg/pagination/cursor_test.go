package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorEncodeDecode(t *testing.T) {
    cursor := &Cursor{
        ID:      uuid.New(),
        StartAt: time.Now().UTC().Round(time.Second),
    }

    encoded := cursor.Encode()
    decoded, err := DecodeCursor(encoded)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if decoded == nil {
        t.Fatalf("decoded cursor is nil")
    }
    if decoded.ID != cursor.ID || !decoded.StartAt.Equal(cursor.StartAt) {
        t.Fatalf("decoded cursor mismatch: %+v", decoded)
    }
}

func TestDecodeCursorInvalid(t *testing.T) {
    if _, err := DecodeCursor("bad!=base64"); err == nil {
        t.Fatalf("expected error for invalid base64")
    }
}

func TestDecodeCursorEmpty(t *testing.T) {
    cursor, err := DecodeCursor("")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cursor != nil {
        t.Fatalf("expected nil cursor, got %+v", cursor)
    }
}

func TestNormalizeLimit(t *testing.T) {
    tests := []struct {
        in   int
        want int
    }{
        {0, DefaultLimit},
        {-10, DefaultLimit},
        {MaxLimit + 1, MaxLimit},
        {50, 50},
    }

    for _, tt := range tests {
        if got := NormalizeLimit(tt.in); got != tt.want {
            t.Fatalf("NormalizeLimit(%d) = %d, want %d", tt.in, got, tt.want)
        }
    }
}
