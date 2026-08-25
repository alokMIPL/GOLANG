package main

import (
	"fmt"
	"maps"
)

// maps => hash, object, dict in different programmign languages.

func main() {

	// Creating map
	m := make(map[string]string)

	// Setting an element
	m["name"] = "golang"
	m["area"] = "backend"

	// Get an element
	fmt.Println(m["name"], m["area"])
	// Output = golang
	// Output = golang backend

	// What happen if we try to access those keys that are not exist in map.
	fmt.Println(m["phone"])
	// Output =
	// If key does not exists in the map then it returns ZERO value.

	// Now check this for BOOLEAN for more confirmation.

	n := make(map[string]int)
	n["roll"] = 12
	fmt.Println(n["roll"], n["kola"])
	// Output = 12 0

	// Now see we initalize roll in mpa not kola but wehn we access these two value, then we get 12 and 0 in map.
	// kola is not present in map so it give 0 because our KEY-VALUE pair is string-int.

	// To get length of a MAP.
	fmt.Println(len(m), len(n))
	// Output = 2 1

	// How to Print whole map not element wise.
	fmt.Println(m)
	// Output = map[area:backend name:golang]

	// Delete ELEMENT for MAP
	delete(m, "area")
	fmt.Println(m)
	// Output= map[name:golang]

	// If we want to clear or empty the map then we use clear() function.

	clear(m)
	fmt.Println(m)
	// Output = map[] empty map.

	// Now we can declare map by another method not using make() function.

	shop := map[string]int{"price": 40, "phones": 3}
	fmt.Println(shop)
	// Output = map[phones:3 price:40]

	// To get any element form map or to check that it is available or not or any conditional things then we use this.

	v, ok := shop["phones"]
	fmt.Println(v)
	if ok {
		fmt.Println("all ok")
	} else {
		fmt.Println("not ok")
	}

	// Output = 3
	// 					all ok

	// Now we have 2 MAP's and i want to check hese two are equal or not.

	m1 := map[string]int{"price": 40, "phones": 3}
	m2 := map[string]int{"price": 40, "phones": 3}

	fmt.Println(maps.Equal(m1, m2))
	// Output = true

	m3 := map[string]int{"price": 140, "phones": 3}
	m4 := map[string]int{"price": 40, "phones": 3}
	fmt.Println(maps.Equal(m3, m4))
	// Output = false

}
