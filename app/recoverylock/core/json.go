package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// StrictJSON decodes one JSON value and refuses duplicate object keys at any
// depth and any data after the value. Numbers are kept as json.Number.
func StrictJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	v, err := strictValue(dec, 0)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("trailing data after the JSON value")
	}
	return v, nil
}

const maxJSONDepth = 32

func strictValue(dec *json.Decoder, depth int) (any, error) {
	if depth > maxJSONDepth {
		return nil, errors.New("JSON nested too deeply")
	}
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch d := t.(type) {
	case json.Delim:
		switch d {
		case '{':
			m := map[string]any{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k, ok := kt.(string)
				if !ok {
					return nil, errors.New("object key is not a string")
				}
				if _, dup := m[k]; dup {
					return nil, fmt.Errorf("duplicate JSON key %q", k)
				}
				x, err := strictValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				m[k] = x
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return m, nil
		case '[':
			var a []any
			for dec.More() {
				x, err := strictValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				a = append(a, x)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return a, nil
		}
		return nil, fmt.Errorf("unexpected %v", d)
	default:
		return t, nil
	}
}
