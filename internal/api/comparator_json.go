package api

import (
	"encoding/json"
	"reflect"
	"strings"
)

// marshalOmitZero marshals a struct to JSON, omitting fields whose values are
// zero (nil pointers, nil slices, zero numbers, empty strings, etc.).
//
// genqlient generates comparator input types without omitempty JSON tags.
// When only some fields are set (e.g. Nin), unset fields serialize as null,
// which the Linear API interprets as conflicting constraints. This helper
// ensures only non-zero fields appear in the JSON payload.
func marshalOmitZero(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	m := make(map[string]any, rt.NumField())
	for i := range rt.NumField() {
		fv := rv.Field(i)
		if fv.IsZero() {
			continue
		}
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		m[name] = fv.Interface()
	}
	return json.Marshal(m)
}

func (s StringComparator) MarshalJSON() ([]byte, error)         { return marshalOmitZero(s) }
func (s NullableStringComparator) MarshalJSON() ([]byte, error) { return marshalOmitZero(s) }
func (n NumberComparator) MarshalJSON() ([]byte, error)         { return marshalOmitZero(n) }
func (n NullableNumberComparator) MarshalJSON() ([]byte, error) { return marshalOmitZero(n) }
func (b BooleanComparator) MarshalJSON() ([]byte, error)        { return marshalOmitZero(b) }
