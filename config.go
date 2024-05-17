package main

type User struct {
	Username string
	Password string
	Targets  map[string]string
}
type config struct {
	Secret string
	Port   int
	Users  map[string]User
}

func (config *config) validateUser(username, password string) bool {
	if p, ok := config.Users[username]; ok {
		return p.Password == password
	}
	return false
}
