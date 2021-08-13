package main

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestVariable(t *testing.T) {
	array:=[]string{"1","2","3"}

	for i:=0;i<100;i++{
		offset:=rand.Intn(len(array))
		fmt.Println(offset)
	}
}
