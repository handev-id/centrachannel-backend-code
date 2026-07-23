package tenantutils

import "encoding/json"

// DeepCopyJSON creates a deep copy of any JSON-serializable value via marshal/unmarshal.
func DeepCopyJSON[T any](src *T) (*T, error) {
	if src == nil {
		return nil, nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst T
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// StripSettings removes sensitive keys from settings JSON.
func StripSettings(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return raw
	}

	delete(settings, "channel_configuration")

	result, err := json.Marshal(settings)
	if err != nil {
		return raw
	}
	return result
}