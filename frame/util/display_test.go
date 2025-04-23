package util

import (
	"fmt"
	"github/beijian128/micius/proto/gameconf"
	"testing"
)

func TestDisplay(t *testing.T) {
	type EventManager struct {
		e []int32
	}
	type User struct {
		name string
		id   uint64
		em   *EventManager
	}
	t.Log(Display(&User{
		name: "Mike",
		id:   123,
		em:   &EventManager{e: []int32{1, 2, 3, 4, 5}},
	}))

}

func TestDisplay1(t *testing.T) {
	type Address struct {
		Street  string
		City    string
		Country string
	}

	type Person struct {
		Name      string
		Age       int
		Address   *Address
		Interests []string
	}
	p := Person{
		Name: "Alice",
		Age:  30,
		Address: &Address{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
		},
		Interests: []string{"Reading", "Traveling", "Gardening"},
	}

	t.Log(Display(p))
	t.Log(fmt.Sprint(p))
	t.Log(fmt.Sprintf("%#v", p))
}

func TestDisplay2(t *testing.T) {

	type Address struct {
		Street  string
		City    string
		Country string
	}

	type Person struct {
		Name      string
		Age       int
		Address   *Address
		Interests []string
	}
	people := []Person{
		{
			Name: "Alice",
			Age:  30,
			Address: &Address{
				Street:  "123 Main St",
				City:    "New York",
				Country: "USA",
			},
			Interests: []string{"Reading", "Traveling", "Gardening"},
		},
		{
			Name: "Bob",
			Age:  25,
			Address: &Address{
				Street: "456 Elm St",
			},
			Interests: []string{"Cooking", "Photography"},
		},
		{
			Name: "Charlie",
			Age:  40,
			Address: &Address{
				City:    "London",
				Country: "UK",
			},
			Interests: []string{"Sports", "Music"},
		},
	}
	t.Log(Display(people))
	t.Log(people)
}

func TestDisplay3(t *testing.T) {

	msg := []*gameconf.PropPack{{PropID: 20001, Num: 1}, {PropID: 20001, Num: 1}, {PropID: 20001, Num: 1}}
	t.Log(Display(msg))
}
