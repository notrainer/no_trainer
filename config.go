package main

type Target struct {
	Name string
	Icon string
	Url  string
}

type User struct {
	Username string
	Password string
	Targets  []*Target
}

type Config struct {
	Secret string
	Port   int
	Users  []*User
}

func (config *Config) getUser(username string) *User {
	for _, u := range config.Users {
		if u.Username == username {
			return u
		}
	}
	return nil
}

func (config *Config) validateUser(username, password string) bool {
	user := config.getUser(username)
	return user != nil && user.Password == password
}
