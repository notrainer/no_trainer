package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"

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

func startServer(config Config) {

	r := gin.Default()
	loadTemplates(r)

	r.GET("/", func(c *gin.Context) {
		tokenString, err := c.Cookie("jwt")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		claims, err := ValidateJWT(tokenString, config.Secret)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		userName := claims["user"].(string)
		user := config.getUser(userName)
		if user == nil {
			c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "User not found"})
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
		fmt.Printf("Targets: %+v\n", targets)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"User":    user,
			"Targets": targets,
		})
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
		c.SetCookie("jwt", jwtToken, 60, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})

	r.Run(fmt.Sprintf(":%d", config.Port))
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

	fmt.Printf("Config: %+v\n", config)

	startServer(config)
}
