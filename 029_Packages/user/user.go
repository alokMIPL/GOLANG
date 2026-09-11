package user

// Same thing if we declare with small alphabrt lke "user" then only used for this package auth.
// But if we use "User" then we can use this as global on in whole project.

type User struct {
	Email string
	Name  string
}
