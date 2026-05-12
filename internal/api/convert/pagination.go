package convert

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// PageCursor is the opaque token used for cursor-based pagination.
type PageCursor struct {
	CreatedAt string `json:"c"`
	ID        string `json:"i"`
}

// EncodeCursor base64-encodes a PageCursor into a page token string.
func EncodeCursor(createdAt, id string) string {
	b, _ := json.Marshal(PageCursor{CreatedAt: createdAt, ID: id})
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor parses a page token string back to a PageCursor.
// Returns an empty cursor (nil cursors) when token is empty.
func DecodeCursor(token string) (createdAt *string, id *string, err error) {
	if token == "" {
		return nil, nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid page token")
	}
	var c PageCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, nil, fmt.Errorf("invalid page token")
	}
	if c.CreatedAt == "" || c.ID == "" {
		return nil, nil, fmt.Errorf("invalid page token")
	}
	return &c.CreatedAt, &c.ID, nil
}

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// PageSizeOrDefault returns the requested page size clamped to [1, 100].
// Falls back to DefaultPageSize when size is 0.
func PageSizeOrDefault(size uint32) int64 {
	if size == 0 {
		return DefaultPageSize
	}
	if size > 100 {
		return 100
	}
	return int64(size)
}
