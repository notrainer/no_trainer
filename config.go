package main

type Target struct {
	Name       string
	Icon       string
	Url        string
	SSHCommand string `yaml:"ssh_command" json:"ssh_command"`
	SSHHostKey string `yaml:"ssh_host_key" json:"ssh_host_key"`
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
