package lojban

import "math/rand"

func GetRandomID(length int) string {
	var id string

	for i := 0; i < length; i++ {
		id += "noparecivomuxazebiso"[rand.Intn(10)*2:][:2]
	}

	return id
}
