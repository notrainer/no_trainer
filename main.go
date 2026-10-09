package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var templates embed.FS

//go:embed dist/*
var dist embed.FS

func loadTemplates(r *gin.Engine) {
	templateFiles := []string{
		"templates/*.html",
		"templates/partials/*.html",
	}

	subFS, err := fs.Sub(dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/assets", http.FS(subFS))

	tmpl := template.New("")

	for _, pattern := range templateFiles {
		_, err := tmpl.ParseFS(templates, pattern)
		if err != nil {
			log.Fatalf("Failed to parse %s: %v", pattern, err)
		}
	}

	r.SetHTMLTemplate(tmpl)
}

type Login struct {
	User     string `form:"username" json:"user" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func authenticatedUser(c *gin.Context, config Config) *User {
	tokenString, err := c.Cookie("jwt")
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}
	claims, err := ValidateJWT(tokenString, config.Secret)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}
	username, ok := claims["user"].(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}
	user := config.getUser(username)
	if user == nil {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "User not found"})
	}
	return user
}

func setupRouter(config Config) *gin.Engine {

	r := gin.Default()
	loadTemplates(r)

	r.GET("/", func(c *gin.Context) {
		user := authenticatedUser(c, config)
		if user == nil {
			return
		}
		var targets []*Target
		for _, target := range user.Targets {
			_, err := url.Parse(target.Url)
			if err != nil {
				log.Printf("Failed to parse URL: %v", err)
				continue
			}
			targets = append(targets, target)
		}
		c.HTML(http.StatusOK, "index.html", gin.H{
			"User":    user,
			"Targets": targets,
		})
	})

	r.GET("/materials/:name", func(c *gin.Context) {
		if authenticatedUser(c, config) == nil {
			return
		}
		name := c.Param("name")
		switch name {
		case "nmap-training.pdf", "nmap-day1.pdf", "nmap-day2.pdf", "nmap-day3.pdf":
			c.Header("Cache-Control", "private, no-store")
			c.File(filepath.Join("/materials", name))
		default:
			c.Status(http.StatusNotFound)
		}
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	r.POST("/login", func(c *gin.Context) {
		var form Login
		if err := c.ShouldBind(&form); err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error()})
			return
		}

		if !config.validateUser(form.User, form.Password) {
			c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Invalid username or password"})
			return
		}
		jwtToken, err := GenerateJWT(form.User, config.Secret)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to generate JWT"})
			fmt.Println(err)
			return
		}
		c.SetCookie("jwt", jwtToken, int(sessionLifetime.Seconds()), "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})

	return r
}

func startServer(config Config) {
	if err := setupRouter(config).Run(fmt.Sprintf(":%d", config.Port)); err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	godotenv.Load()

	pflag.IntP("port", "p", 8080, "Port to listen on")
	pflag.StringP("secret", "s", "changeme", "JWT secret")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetEnvPrefix("no_trainer")
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.ReadInConfig()

	config := Config{}
	viper.Unmarshal(&config)

	usersString := viper.GetString("users")
	if usersString != "" {
		yaml.Unmarshal([]byte(usersString), &config.Users)
	}
	targetsString := viper.GetString("targets")
	var globalTargets []*Target
	if targetsString != "" {
		yaml.Unmarshal([]byte(targetsString), &globalTargets)
	}

	for _, user := range config.Users {
		user.Targets = append(globalTargets, user.Targets...)
	}

	startServer(config)
}
