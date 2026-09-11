package main

import (
	"fmt"

	"github.com/alok/podcast/auth"
)

// go mod init github.com/alok/podcast

// GOLANG only compile those file whioch have change not other files.

func main() {
	auth.LoginWithCredentials("alok", "kola")
	session := auth.GetSession()
	fmt.Println(session)
}
