package util

import (
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		Bag  []int
	}

	srcPerson := Person{Name: "Alice", Age: 30, Bag: []int{1, 2, 3, 4, 5}}
	var copiedPerson Person

	err := DeepCopy(&copiedPerson, srcPerson)
	if err != nil {
		t.Fatalf("DeepCopy failed with error: %v", err)
	}

	if !reflect.DeepEqual(srcPerson, copiedPerson) {
		t.Errorf("Copied person does not match the source person")
	}
	t.Log(srcPerson, copiedPerson)
}
