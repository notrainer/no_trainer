package main

type User struct {
	Username string
	Password string
	Clients  []string
}

type Client struct {
	Schema   string
	Target   string
	Username string
	Password string
}

type config struct {
	Secret  []byte
	Port    int
	Users   map[string]User
	Clients map[string]Client
}

func (config *config) validateUser(username, password string) bool {
	if p, ok := config.Users[username]; ok {
		return p.Password == password
	}
	return false
}
