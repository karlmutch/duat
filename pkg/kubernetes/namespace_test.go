package kubernetes

import (
	"testing"
)

func TestSimpleTriming(t *testing.T) {
	inputs := []string{"1", "1-2-3-4", "aa-bb-cc-dd-ee", "aa-bbbb-dd-ee", "aaaa-bbb-ccc", "12345678901"}
	expected := []string{"1", "1-2-3-4", "aa-bb-ee", "aa-bbbb-ee", "aaaa-ccc", "1234567890"}

	for i, aTest := range inputs {
		result := wordTrim(aTest, "-", 10)
		if expected[i] != result {
			t.Error("failure of case", aTest, "got", result, "expected", expected[i])
		}
	}
}

func TestNamespaceTriming(t *testing.T) {
	inputs := []string{"gw-0-9-23-feature-265-zero-length-metadata-reinstated-aaaagmwypak"}
	expected := []string{"gw-0-9-23-feature-265-zero-length-metadata-aaaagmwypak"}

	for i, aTest := range inputs {
		result := TrimNamespace(aTest)
		if expected[i] != result {
			t.Error("failure of case", aTest, "got", result, "expected", expected[i])
		}
	}
}
