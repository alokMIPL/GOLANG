package main

import (
	"fmt"

	"github.com/alok/podcast/auth"
	"github.com/alok/podcast/user"
)

// go mod init github.com/alok/podcast

// GOLANG only compile those file whioch have change not other files.

func main() {
	auth.LoginWithCredentials("alok", "kola")
	session := auth.GetSession()
	fmt.Println("session = ", session)

	user := user.User{
		Email: "john@email.com",
		Name:  "John Deo",
	}

	fmt.Println(user.Email)
	fmt.Println(user.Name)

	// In GOLANG there are various packages but if we want to install third party packages then we can do that also by using thirt part packages library

}
