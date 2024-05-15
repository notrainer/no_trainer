package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

//go:embed templates/*
var templates embed.FS

//go:embed dist/*
var dist embed.FS

func GetTemplateFiles(patterns ...string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

func loadTemplates(r *gin.Engine) {
	templateFiles := []string{
		"templates/*.html",
	}

	if gin.Mode() == gin.ReleaseMode {
		subFS, err := fs.Sub(dist, "dist")
		if err != nil {
			log.Fatal(err)
		}
		r.StaticFS("/assets", http.FS(subFS))

		tmpl := template.New("")

		// Parse files in the specified order
		for _, pattern := range templateFiles {
			_, err := tmpl.ParseFS(templates, pattern)
			if err != nil {
				log.Fatalf("Failed to parse %s: %v", pattern, err)
			}
		}

		r.SetHTMLTemplate(tmpl)
	} else {
		r.Static("/assets", "./dist")

		// Collect all template files
		templates, err := GetTemplateFiles(templateFiles...)
		if err != nil {
			log.Fatal(err)
		}

		// Parse all templates with the custom function map
		tmpl, err := template.New("").ParseFiles(templates...)
		if err != nil {
			log.Fatal(err)
		}

		r.SetHTMLTemplate(tmpl)
	}
}

type Login struct {
	User     string `form:"username" json:"user" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func startServer(config config) {

	templateContext := gin.H{}

	r := gin.Default()
	loadTemplates(r)

	r.GET("/", func(c *gin.Context) {
		tokenString, err := c.Cookie("jwt")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		if _, err := ValidateJWT(tokenString, config.Secret); err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.HTML(http.StatusOK, "index.html", templateContext)
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", templateContext)
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

	pflag.IntP("port", "p", 8080, "Port to listen on")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetEnvPrefix("no_trainer")
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.ReadInConfig()

	config := config{}
	viper.Unmarshal(&config)

	fmt.Printf("Config: %+v\n", config)

	startServer(config)
}
