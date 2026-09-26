package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

func main() {
	url := "https://catfact.ninja/fact"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		return
	}

	// It convert JSON BYTES in GO Structs
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read body failed", err)
		return
	}

	// Optional i do for Testing.
	bodyText := string(bodyBytes)
	fmt.Println("For Testing", bodyText)

	var data CatFactResponse

	// json.Unmarshal(bodyBytes, &data) — try to convert the JSON bytes into the data struct.
	// and store the data in to data variable by the help of `&pointer`.
	// It converts the raw JSON bytes into a typed Go struct

	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		fmt.Println("Json Unmarshal failed.", err)
		return
	}

	// Complex Way
	// if err := json.Unmarshal(bodyBytes, &data); err != nil {
	// 	fmt.Println("Json Unmarshal failed.")
	// 	return
	// }

	fmt.Println(data.Fact, data.Length)

}

/*
Output = A cats field of vision is about 185 degrees. 44
*/
