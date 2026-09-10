package auth

import "fmt"

/*
I have a function loginWithCredentials Now,
1. If Function start with small letter then we only use that particular function on that folder. In this loginWithCredentials() used only on /auth folder.

2. If we want to use loginWithCredentials() on whole project or export outside that folder then we need to declare our function in capital letter LoginWithCredentials()
Now we can use this LoginWithCredentials() anywhere in this project

*/

func loginWithCredentials(username string, password string) {
	fmt.Println("Login user using", username, password)
}
