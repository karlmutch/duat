package kv

import (
	"regexp"
	"strconv"
)

type keyvalser interface {
	Keyvals() []interface{}
}

// flattenFix accepts a keyvals slice and "flattens" it into a slice
// of alternating key/value pairs. See the examples.
//
// flattenFix will also check the result to ensure it is a valid
// slice of key/value pairs. The returned slice is guaranteed
// to have an even number of items, and every item at an even-numbered
// index is guaranteed to be a string.
func flattenFix(keyvals []interface{}) []interface{} {
	// opinionated constants for names of keys
	const (
		keyMsg           = "msg"
		keyError         = "error"
		keyMissingPrefix = "_p"
	)

	// Indicates whether the keyvals slice needs to be flattened.
	// Start with true if it has an odd number of items.
	requiresFlattening := isOdd(len(keyvals))

	// Used for estimating the size of the flattened slice
	// in an attempt to use one memory allocation.
	var estimatedLen int

	// Do the keyvals include a "msg" key. This is not entirely
	// reliable if "msg" is passed as a value, but it is only used
	// as a heuristic for naming missing keys.
	var haveMsg bool

	for i, val := range keyvals {
		switch v := val.(type) {
		case List:
			requiresFlattening = true
			// TODO(jpj): recursively descending into the keyvals
			// will come up with a reasonably length estimate, but
			// for now just double the number of elements in the slice
			// and this is probably accurate enough.
			estimatedLen += len(v)
		case keyvalser:
			// unknown implementation: calling Keyvals could result in
			// additional memory allocation, so use a conservative guess
			requiresFlattening = true
			estimatedLen += 16
		case string:
			if v == keyMsg {
				// Remember that we already have a "msg" key, which
				// will be used for inferring missing key names later.
				haveMsg = true
			}
		default:
			estimatedLen++
			if isEven(i) {
				// Non-string in an even position could mean a missing
				// key name in the list.
				estimatedLen++
				requiresFlattening = true
			}
		}
	}

	if !requiresFlattening {
		// Nothing to do, so return the input unmodified.
		return keyvals
	}

	// The missingKey function is passed recursively to flattening
	// and fixing functions. After flattening and fixing we know if
	// one or more missing keys have been inserted, and then we know
	// to iterate through and name them in order.
	var hasMissingKeys bool
	missingKey := func() interface{} {
		hasMissingKeys = true
		return missingKeyT("MISSING")
	}

	// In most circumstances this output slice will have the
	// required capacity.
	output := make([]interface{}, 0, estimatedLen)

	// Perform the actual flattening and fixing.
	output = flatten(output, keyvals, missingKey)

	// If there were any missing keys inserted, iterate through the
	// list and name them. Doing this last allows the names to be
	// ordered from left to right.
	if hasMissingKeys {
		counter := 0 // used for counting _p1, _p2, etc
		for i, v := range output {
			if _, ok := v.(missingKeyT); ok {
				// assign a name for the missing key, depends on the type
				// of the value associated with the key
				var keyName string
				switch output[i+1].(type) {
				case string:
					if !haveMsg {
						// If there is no 'msg' key, the first string
						// value gets 'msg' as its key.
						haveMsg = true
						keyName = keyMsg
					}
				case error:
					if haveMsg {
						// If there is already a 'msg' key, then an
						// error gets 'error' as the key.
						keyName = keyError
					} else {
						// If there is no 'msg' key, the first error
						// value gets 'msg' as its key.
						haveMsg = true
						keyName = keyMsg
					}
				}
				if keyName == "" {
					// Otherwise, missing keys all have a prefix that is
					// unlikely to clash with others key names, and are
					// numbered from 1.
					counter++
					keyName = keyMissingPrefix + strconv.Itoa(counter)
				}
				output[i] = keyName
			}
		}
	}

	return output
}

// isEven returns true if i is even.
func isEven(i int) bool {
	return (i & 0x01) == 0
}

// isOdd returns true if i is odd.
func isOdd(i int) bool {
	return (i & 0x01) != 0
}

// The missingKeyT type is used as a placeholder for missing keys.
// Once all the missing keys are inserted, they are numbered from left
// to right.
type missingKeyT string

func flatten(
	output []interface{},
	input []interface{},
	missingKeyName func() interface{},
) []interface{} {
	for len(input) > 0 {
		// Process any leading scalars. A scalar is any single value,
		// ie not a keyvalsAppender, keyvalser, keyvalPairer or keyvalMapper.
		// This makes it easier to figure out any missing key names.
		if i := countScalars(input); i > 0 {
			output = flattenScalars(output, input[:i], missingKeyName)
			input = input[i:]
			continue
		}

		// At this point the first item in the input is a keyvalsAppender,
		// keyvalser, keyvalPairer, or keyvalMapper.
		switch v := input[0].(type) {
		case keyvalser:
			// The Keyvals method does not guarantee to return a valid
			// key/value list, so flatten and fix it as if this slice
			// had been passed to the flattenFix function in the first place.
			output = flatten(output, v.Keyvals(), missingKeyName)
		default:
			//panic("cannot happen")
		}

		input = input[1:]
	}

	return output
}

// countScalars returns the count of items in input up to but
// not including the first non-scalar item. A scalar is a single
// value item, ie not a keyvalser.
func countScalars(input []interface{}) int {
	for i := 0; i < len(input); i++ {
		switch input[i].(type) {
		case keyvalser:
			return i
		}
	}
	return len(input)
}

// flattenScalars adjusts a list of items, none of which are keyvalsers.
//
// Ideally the list will have an even number of items, with strings in the
// even indices. If it doesn't, this function will fix it.
func flattenScalars(
	output []interface{},
	input []interface{},
	missingKeyName func() interface{},
) []interface{} {
	for len(input) > 0 {
		var needsFixing bool

		if isOdd(len(input)) {
			needsFixing = true
		} else {
			// check for non-string in an even position
			for i := 0; i < len(input); i += 2 {
				switch input[i].(type) {
				case string, missingKeyT:
					break
				default:
					needsFixing = true
				}
			}
		}

		if !needsFixing {
			output = append(output, input...)
			input = nil
			continue
		}

		// Build a classification of items in the array. This will be used
		// to determine the most likely position of missing key(s).
		// TODO(jpj): this could be allocated from a sync.Pool
		type classificationT byte
		const (
			stringKey classificationT = iota
			stringPossibleKey
			stringValue
			errorVar
			otherType
		)
		classifications := make([]classificationT, len(input))

		for i := 0; i < len(input); i++ {
			switch v := input[i].(type) {
			case string:
				if _, ok := knownKeys[v]; ok {
					classifications[i] = stringKey
				} else if possibleKeyRE.MatchString(v) {
					classifications[i] = stringPossibleKey
				} else {
					classifications[i] = stringValue
				}
			case missingKeyT:
				classifications[i] = stringKey
			default:
				classifications[i] = otherType
			}
		}

		if len(input) == 1 {
			// Only one parameter, give it a key name. If it is a string it might
			// be the 'msg' parameter.
			output = append(output, missingKeyName())
			output = append(output, input[0])
			input = nil
			continue
		}

		// Function to insert a key before an item that is either unlikely
		// or impossible to be a key. Returns true if something was inserted.
		// Note that this function assumes that there are at least two items
		// in the slice, which is guaranteed at this point.
		insertKeyFromBack := func(c classificationT) bool {
			// Start at the second last item
			for i := len(input) - 2; i > 0; i -= 2 {
				if classifications[i] == c {
					if isEven(len(input)) {
						input = insertKeyAt(input, i, missingKeyName())
					} else {
						input = insertKeyAt(input, i+1, missingKeyName())
					}
					return true
				}
			}
			return false
		}
		if insertKeyFromBack(otherType) {
			continue
		}
		if insertKeyFromBack(errorVar) {
			continue
		}
		if insertKeyFromBack(stringValue) {
			continue
		}
		insertKeyFromFront := func(c classificationT) bool {
			for i := 0; i < len(input); i += 2 {
				if classifications[i] == c {
					input = insertKeyAt(input, i, missingKeyName())
					return true
				}
			}
			return false
		}
		if insertKeyFromFront(otherType) {
			continue
		}
		if insertKeyFromFront(errorVar) {
			continue
		}
		if insertKeyFromFront(stringValue) {
			continue
		}
		if insertKeyFromFront(stringPossibleKey) {
			continue
		}
		input = insertKeyAt(input, 0, missingKeyName())
	}
	return output
}

// possibleKeyRE is a regexp for deciding if a string value is likely
// to be a key rather than a value.
var possibleKeyRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

func insertKeyAt(input []interface{}, index int, keyName interface{}) []interface{} {
	newInput := make([]interface{}, 0, len(input)+1)
	if index > 0 {
		newInput = append(newInput, input[0:index]...)
	}
	newInput = append(newInput, keyName)
	newInput = append(newInput, input[index:]...)
	return newInput
}

// this could be public and configurable
var knownKeys = map[string]struct{}{
	"msg":   {},
	"level": {},
	"id":    {},
}
