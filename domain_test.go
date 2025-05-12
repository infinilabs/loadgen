// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Loadgen is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"infini.sh/framework/lib/fasttemplate"
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
