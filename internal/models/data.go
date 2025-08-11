package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDataNotFound = errors.New("data not found")
)

type DataType string

const (
	DataTypeCredentials DataType = "credentials"
	DataTypeText        DataType = "text"
	DataTypeBinary      DataType = "binary"
	DataTypeCard        DataType = "card"
)

type DataItem struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Type      DataType   `json:"type" db:"type"`
	Name      string     `json:"name" db:"name"`
	Data      []byte     `json:"data" db:"data"`
	Metadata  []byte     `json:"metadata" db:"metadata"`
	Version   int64      `json:"version" db:"version"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	URL      string `json:"url,omitempty"`
}

type TextData struct {
	Content string `json:"content"`
}

type BinaryData struct {
	Content  []byte `json:"content"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type CardData struct {
	Number     string `json:"number"`
	Holder     string `json:"holder"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
	Bank       string `json:"bank,omitempty"`
}
