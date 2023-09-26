package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

const ShortURLLength = 12

var LetterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type LinkShortener struct {
	mutex sync.Mutex
	links map[string]string
}

func NewLinkShortener() *LinkShortener {
	return &LinkShortener{links: make(map[string]string)}
}

func generateShortURL() string {
	b := make([]rune, ShortURLLength)
	for i := range b {
		b[i] = LetterRunes[rand.Intn(len(LetterRunes))]
	}
	return string(b)
}

func (ls *LinkShortener) Shorten(longURL, customName string) (string, error) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if longURL == "" {
		return "", fmt.Errorf("no URL specified")
	}

	if customName != "" {
		if _, exists := ls.links[customName]; exists {
			return "", fmt.Errorf("custom name %s already exists", customName)
		}
		ls.links[customName] = longURL
		return customName, nil
	}

	shortURL := generateShortURL()
	for {
		if _, exists := ls.links[shortURL]; !exists {
			ls.links[shortURL] = longURL
			fmt.Println(shortURL, longURL)
			return shortURL, nil
		}
		shortURL = generateShortURL()
	}
}

func (ls *LinkShortener) Expand(shortURL string) (string, error) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	if shortURL == "" {
		return "", fmt.Errorf("no URL specified")
	}

	if longURL, ok := ls.links[shortURL]; ok {
		return longURL, nil
	}

	return "", fmt.Errorf("URL not found")
}

func main() {
	LinkShortener := NewLinkShortener()
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.POST("/shorten", func(c *gin.Context) {
		longURL := c.PostForm("longURL")
		customName := c.PostForm("customName")
		fmt.Println(longURL, customName)
		go func() {
			if shortURL, err := LinkShortener.Shorten(longURL, customName); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
			}
		}()
	})

	router.POST("/expand", func(c *gin.Context) {
		shortURL := c.PostForm("shortURL")
		go func() {
			if longURL, err := LinkShortener.Expand(shortURL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"longURL": longURL})
			}
		}()
	})

	router.GET("/:shortURL", func(c *gin.Context) {
		shortURL := c.Param("shortURL")
		go func() {
			if longURL, err := LinkShortener.Expand(shortURL); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.Redirect(http.StatusFound, longURL)
			}
		}()
	})
	router.Run(":8080")
}
