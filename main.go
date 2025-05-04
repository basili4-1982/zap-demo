package main

import (
	"fmt"
	"slices"
)

var m1 = map[string]int{
	"a": 1,
	"b": 2,
	"c": 3,
	"x": 4,
}
var m2 = map[string]int{
	"z": 80,
	"a": 34,
	"b": 25,
	"c": 26,
}

func main() {
	for k, v := range m1 {
		fmt.Println(k, v)
	}

	fmt.Println("----------------")

	for k, v := range m2 {
		fmt.Println(k, v)
	}

	keys := make([]string, 0, len(m1)+len(m2))

	exist := make(map[string]struct{})

	for k, _ := range m1 {
		if _, ok := exist[k]; !ok {
			keys = append(keys, k)
			exist[k] = struct{}{}
		}
	}

	for k, _ := range m2 {
		if _, ok := exist[k]; !ok {
			keys = append(keys, k)
			exist[k] = struct{}{}
		}
	}

	fmt.Println("----------------")

	fmt.Println(keys)

	slices.Sort(keys)

	fmt.Println(keys)

	fmt.Println("----------------")

	for _, k := range keys {
		if v, ok := m1[k]; ok {
			fmt.Println(k, v)
		} else {
			fmt.Println(k, "-")
		}
		if v, ok := m2[k]; ok {
			fmt.Println(k, v)
		} else {
			fmt.Println(k, "-")
		}
	}

}
