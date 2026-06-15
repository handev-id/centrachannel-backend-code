package utils

import (
	"database/sql"
	"encoding/json"
)

type NullableTime struct {
	sql.NullTime
}

func (nt NullableTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

func (nt *NullableTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	nt.Valid = true
	return json.Unmarshal(data, &nt.Time)
}
