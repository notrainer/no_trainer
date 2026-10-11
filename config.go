package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

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
	Targets  TargetList
}

type TargetList []*Target

func (targets *TargetList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		var list []*Target
		if err := node.Decode(&list); err != nil {
			return err
		}
		*targets = list
		return nil
	}
	var urls map[string]string
	if err := node.Decode(&urls); err != nil {
		return err
	}
	*targets = nil
	for _, name := range slices.Sorted(maps.Keys(urls)) {
		*targets = append(*targets, &Target{Name: name, Icon: "vnc-icon", Url: urls[name]})
	}
	return nil
}

type Config struct {
	Secret         string
	Port           int
	Users          []*User
	NmapVisualizer bool
}

// Environment variables contain YAML/JSON strings; config files already contain
// decoded values. Normalize both before decoding structured configuration.
func configYAML(v *viper.Viper, key string) ([]byte, error) {
	value := v.Get(key)
	if text, ok := value.(string); ok {
		return []byte(text), nil
	}
	return yaml.Marshal(value)
}

func loadConfig(v *viper.Viper) (Config, error) {
	config := Config{Port: v.GetInt("port"), Secret: v.GetString("secret")}
	if config.Port < 1 || config.Port > 65535 {
		return Config{}, fmt.Errorf("port must be between 1 and 65535")
	}
	if v.IsSet("nmap_visualizer") {
		enabled, err := strconv.ParseBool(v.GetString("nmap_visualizer"))
		if err != nil {
			return Config{}, fmt.Errorf("nmap_visualizer must be true or false")
		}
		config.NmapVisualizer = enabled
	}
	data, err := configYAML(v, "users")
	if err != nil {
		return Config{}, fmt.Errorf("cannot decode users configuration")
	}
	var document yaml.Node
	if yaml.Unmarshal(data, &document) != nil {
		return Config{}, fmt.Errorf("users must contain valid YAML or JSON")
	}
	if len(document.Content) > 0 && document.Content[0].Tag != "!!null" {
		node := document.Content[0]
		switch node.Kind {
		case yaml.SequenceNode:
			if node.Decode(&config.Users) != nil {
				return Config{}, fmt.Errorf("users list contains an invalid entry")
			}
		case yaml.MappingNode:
			var users map[string]*User
			if node.Decode(&users) != nil {
				return Config{}, fmt.Errorf("users object must map usernames to user objects")
			}
			for _, username := range slices.Sorted(maps.Keys(users)) {
				user := users[username]
				if user == nil || (user.Username != "" && user.Username != username) {
					return Config{}, fmt.Errorf("users object contains an empty entry or conflicting username")
				}
				user.Username = username
				config.Users = append(config.Users, user)
			}
		default:
			return Config{}, fmt.Errorf("users must be a list or an object keyed by username")
		}
	}

	data, err = configYAML(v, "targets")
	var targets TargetList
	if err != nil || yaml.Unmarshal(data, &targets) != nil {
		return Config{}, fmt.Errorf("targets must contain a valid YAML or JSON list or name-to-URL object")
	}
	for _, user := range config.Users {
		if user == nil || user.Username == "" || user.Password == "" {
			return Config{}, fmt.Errorf("each user must have a username and password")
		}
		user.Targets = append(targets, user.Targets...)
		for _, target := range user.Targets {
			if target == nil {
				return Config{}, fmt.Errorf("targets cannot contain empty entries")
			}
		}
	}
	return config, nil
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
