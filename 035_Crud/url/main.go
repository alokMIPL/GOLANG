package main

import (
	"fmt"
	"net/url"
)

func main() {
	fmt.Println("Url")
	myURL := "https://www.example.com/path/to/resource?query=param#fragment"

	fmt.Println("Type of URL: %T\n", myURL)

	parsedURL, err := url.Parse(myURL)

	if err != nil {
		fmt.Println("Can't parse URL", err)
	}

	fmt.Println("Parsed URL:", parsedURL)

	// Some Other Things in URL
	fmt.Println("Scheme:", parsedURL.Scheme)
	fmt.Println("Host:", parsedURL.Host)
	fmt.Println("Path:", parsedURL.Path)
	fmt.Println("RawQuery:", parsedURL.RawQuery)
	fmt.Println("Fragment:", parsedURL.Fragment)
}
