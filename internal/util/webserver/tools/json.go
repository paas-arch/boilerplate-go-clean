package tools

import "encoding/json"

var JSON = &jsonWrapper{}

type jsonWrapper struct{}

func (j *jsonWrapper) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (j *jsonWrapper) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
