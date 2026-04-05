package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice handles []string <-> JSON text in the database.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(s)
	return string(data), err
}

func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("cannot scan %T into StringSlice", value)
	}
	if len(bytes) == 0 {
		*s = StringSlice{}
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// CastSlice handles []Cast <-> JSON text in the database.
type CastSlice []Cast

func (c CastSlice) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(c)
	return string(data), err
}

func (c *CastSlice) Scan(value any) error {
	if value == nil {
		*c = CastSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("cannot scan %T into CastSlice", value)
	}
	if len(bytes) == 0 {
		*c = CastSlice{}
		return nil
	}
	return json.Unmarshal(bytes, c)
}
