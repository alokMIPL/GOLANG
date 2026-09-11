package auth

// now see this extractSession() is private but GetSession() is not private.

func extractSession() string {
	return "Loggedin"
}

func GetSession() string {
	return extractSession()
}
