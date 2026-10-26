package main

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func testConfig() *viper.Viper {
	v := viper.New()
	v.SetDefault("port", 8080)
	v.SetDefault("secret", "test-secret")
	v.SetEnvPrefix("no_trainer")
	v.AutomaticEnv()
	return v
}

func TestConfigUserFormatsAndEnvironment(t *testing.T) {
	for _, users := range []string{
		`[{"username":"admin","password":"admin","targets":[{"name":"private","url":"/login"}]}]`,
		`{"admin":{"password":"admin","targets":[{"name":"private","url":"/login"}]}}`,
		`{"admin":{"username":"admin","password":"admin","targets":{"private":"/login"}}}`,
	} {
		t.Run(users[:1], func(t *testing.T) {
			t.Setenv("NO_TRAINER_USERS", users)
			t.Setenv("NO_TRAINER_TARGETS", `[{"name":"global","url":"/login"}]`)
			t.Setenv("NO_TRAINER_PORT", "8181")
			t.Setenv("NO_TRAINER_SECRET", "environment-secret")
			v := testConfig()
			v.SetConfigType("yaml")
			if err := v.ReadConfig(strings.NewReader("users:\n  - username: other\n    password: other\n")); err != nil {
				t.Fatal(err)
			}
			config, err := loadConfig(v)
			if err != nil {
				t.Fatal(err)
			}
			if !config.validateUser("admin", "admin") || config.validateUser("other", "other") {
				t.Fatal("environment users did not override file users")
			}
			if config.Port != 8181 || config.Secret != "environment-secret" {
				t.Fatal("environment scalars were not loaded")
			}
			if len(config.Users[0].Targets) != 2 || config.Users[0].Targets[0].Name != "global" {
				t.Fatal("global and user targets were not merged")
			}
		})
	}
}

func TestConfigFileUsers(t *testing.T) {
	t.Setenv("NO_TRAINER_USERS", "")
	t.Setenv("NO_TRAINER_TARGETS", "")
	v := testConfig()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("users:\n  - username: admin\n    password: admin\n")); err != nil {
		t.Fatal(err)
	}
	config, err := loadConfig(v)
	if err != nil || !config.validateUser("admin", "admin") {
		t.Fatalf("file users did not load: %v", err)
	}
}

func TestWebluaConfiguration(t *testing.T) {
	config, err := loadConfig(testConfig())
	if err != nil || config.Weblua {
		t.Fatalf("Weblua must default to disabled: %v", err)
	}
	for _, test := range []struct {
		value   string
		enabled bool
		invalid bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"invalid", false, true},
	} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("NO_TRAINER_WEBLUA", test.value)
			v := testConfig()
			v.SetConfigType("yaml")
			if err := v.ReadConfig(strings.NewReader("weblua: true\n")); err != nil {
				t.Fatal(err)
			}
			config, err := loadConfig(v)
			if (err != nil) != test.invalid || config.Weblua != test.enabled {
				t.Fatalf("unexpected Weblua setting: enabled=%v, err=%v", config.Weblua, err)
			}
		})
	}
}

func TestConfigRejectsInvalidUsersWithoutLeakingCredentials(t *testing.T) {
	for _, users := range []string{`{"admin":`, `[{"username":"admin","password":[]}]`, `{"admin":null}`, `"private-secret"`} {
		t.Setenv("NO_TRAINER_USERS", users)
		_, err := loadConfig(testConfig())
		if err == nil {
			t.Fatal("invalid users were accepted")
		}
		if strings.Contains(err.Error(), "private-secret") {
			t.Fatal("configuration error leaked a secret")
		}
	}
}
