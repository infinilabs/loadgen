package main

import (
	"fmt"
	"github.com/valyala/fasttemplate"
	"io"
	"log"
	"math/rand"
	"testing"
)

func TestVariable(t *testing.T) {
	array := []string{"1", "2", "3"}

	for i := 0; i < 100; i++ {
		offset := rand.Intn(len(array))
		fmt.Println(offset)
	}
}

func TestTemplate(t1 *testing.T) {
	template := "Hello, $[[user]]! You won $[[prize]]!!! $[[foobar]]"
	t, err := fasttemplate.NewTemplate(template, "$[[", "]]")
	if err != nil {
		log.Fatalf("unexpected error when parsing template: %s", err)
	}
	s := t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "user":
			return w.Write([]byte("John"))
		case "prize":
			return w.Write([]byte("$100500"))
		default:

			return w.Write([]byte(fmt.Sprintf("[unknown tag %q]", tag)))
		}
	})
	fmt.Printf("%s", s)

}
