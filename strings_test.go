package main
import (
	"reflect"
	"testing"
)

func TestFieldsN(t *testing.T) {
	tests := []struct {
		s string
		n int
		want []string
	}{
		{ "", 1, []string{"^", "", ""} },
		{ "^", 2, []string{"^^^", "", ""} },
		{ " ", 2, []string{"^", "", ""} },
		{ "Т показать", 2, []string{"^Т", "показать", "показать"} },
		{ "Т взять вешалк%", 2, []string{"^Т", "взять", "взять вешалк%"} },
		{ "Л -3001", 1, []string{"^Л", "-3001", "-3001"} },
		{ "Ложь.", 1, []string{"^Ложь.", "", ""} },
		{ " ^взя = взять, прибрать", 2, []string{"^^взя", "=", "взять, прибрать"} },
	}
	for _, test := range tests {
		got := fieldsN(test.s, test.n)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("fields(%q, %v): wanted %q, got %q\n", test.s, test.n, test.want, got)
		}
	}
}
