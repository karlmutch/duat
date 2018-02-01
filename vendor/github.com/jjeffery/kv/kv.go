package kv

import (
	"bytes"
	"sort"
)

// The keyvalser interface returns a slice of alternating keys
// and values.
type keyvalser interface {
	Keyvals() []interface{}
}

// The keyvalPairer interface returns a single key/value pair.
// Internal interface used to reduce memory allocations.
type keyvalPairer interface {
	keyvalPair() (key string, value interface{})
}

// The keyvalsAppender interface is used for appending key/value pairs.
// This is an internal interface: the promise is that it will only
// append valid key/value pairs.
//
// This internal interface can really be removed, but at time of writing
// I'm not sure that the keyvalPairer and keyvalMapper interfaces are
// going to stay. If they do get removed, then the keyvalsAppender provides
// a sneaky way to reduce memory allocations for Pair and Map types.
// So... remove the keyvalsAppender interface if keyvalPairer and keyvalMapper
// do in fact prove their worth and are made permanent.
type keyvalsAppender interface {
	appendKeyvals(keyvals []interface{}) []interface{}
}

// List is a slice of alternating keys and values.
type List []interface{}

// Keyvals returns the list cast as []interface{}.
// It implements the keyvalser interface described in the package summary.
func (l List) Keyvals() []interface{} {
	return []interface{}(l)
}

// String returns a string representation of the key/value pairs in
// logfmt format: "key1=value1 key2=value2  ...".
func (l List) String() string {
	var buf bytes.Buffer
	l.writeToBuffer(&buf)
	return buf.String()
}

// MarshalText implements the TextMarshaler interface.
func (l List) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	l.writeToBuffer(&buf)
	return buf.Bytes(), nil
}

func (l List) writeToBuffer(buf *bytes.Buffer) {
	fl := Flatten(l)
	for i := 0; i < len(fl); i += 2 {
		k := fl[i]
		v := fl[i+1]
		writeKeyValue(buf, k, v)
	}
}

// Pair represents a single key/value pair.
type Pair struct {
	Key   string
	Value interface{}
}

// P returns a key/value pair. The following alternatives are equivalent:
//  kv.Pair{key, value}
//  kv.P(key, value)
// The second alternative is slightly less typing, and avoids
// the following go vet warning:
//  composite literal uses unkeyed fields
func P(key string, value interface{}) Pair {
	return Pair{
		Key:   key,
		Value: value,
	}
}

// Keyvals returns the pair's key and value as a slice of interface{}.
// It implements the keyvalser interface described in the package summary.
func (p Pair) Keyvals() []interface{} {
	return []interface{}{p.Key, p.Value}
}

// keyvalPair returns the pair's key and value. This implements
// the keyvalsPairer interface.
func (p Pair) keyvalPair() (key string, value interface{}) {
	return p.Key, p.Value
}

// String returns a string representation of the key and value in
// logfmt format: "key=value".
func (p Pair) String() string {
	var buf bytes.Buffer
	writeKeyValue(&buf, p.Key, p.Value)
	return buf.String()
}

// MarshalText implements the TextMarshaler interface.
func (p Pair) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	writeKeyValue(&buf, p.Key, p.Value)
	return buf.Bytes(), nil
}

func (p Pair) appendKeyvals(keyvals []interface{}) []interface{} {
	return append(keyvals, p.Key, p.Value)
}

// Map is a map of keys to values.
//
// Note that when a map is appended to a keyvals list of alternating
// keys and values, there is no guarantee of the order that the key/value
// pairs will be appended.
type Map map[string]interface{}

// Keyvals returns the contents of the map as a list of alternating
// key/value pairs. It implements the keyvalser interface described
// in the package summary.
// The key/value pairs are sorted by key.
func (m Map) Keyvals() []interface{} {
	keyvals := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		keyvals = append(keyvals, k, v)
	}
	kvSorter(keyvals).Sort()
	return keyvals
}

func (m Map) appendKeyvals(keyvals []interface{}) []interface{} {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		keyvals = append(keyvals, key, m[key])
	}
	return keyvals
}

// String returns a string representation of the key/value pairs in
// logfmt format: "key1=value1 key2=value2  ...".
// The key value pairs are printed sorted by the key.
func (m Map) String() string {
	var buf bytes.Buffer
	m.writeToBuffer(&buf)
	return buf.String()
}

// MarshalText implements the TextMarshaler interface.
// The key value pairs are marshaled sorted by the key.
func (m Map) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	m.writeToBuffer(&buf)
	return buf.Bytes(), nil
}

func (m Map) writeToBuffer(buf *bytes.Buffer) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeKeyValue(buf, k, m[k])
	}
}

// kvSorter implements sort.Interface. It assumes that every
// element at an even-numbered index is a string and will panic
// if this is not the case.
type kvSorter []interface{}

func (kvs kvSorter) Len() int {
	return len(kvs) / 2
}

func (kvs kvSorter) Less(i, j int) bool {
	key1 := kvs[i*2].(string)
	key2 := kvs[j*2].(string)
	return key1 < key2
}

func (kvs kvSorter) Swap(i, j int) {
	kvs[i*2], kvs[i*2+1], kvs[j*2], kvs[j*2+1] = kvs[j*2], kvs[j*2+1], kvs[i*2], kvs[i*2+1]
}

func (kvs kvSorter) Sort() {
	sort.Sort(kvs)
}
