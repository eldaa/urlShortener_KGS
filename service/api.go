package service

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/elahe-dastan/urlShortener/config"
	"github.com/elahe-dastan/urlShortener/metric"
	"github.com/elahe-dastan/urlShortener/request"
	"github.com/elahe-dastan/urlShortener/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type API struct {
	Map      store.Mapping
	ShortURL store.ShortURL
	URL      config.ShortURL
}

func (a API) Run(cfg config.LogFile) {
	e := echo.New()
	e.POST("/urls", a.Mapping)
	e.GET("/redirect/:shortURL", a.Redirect)

	go func() {
		metric.Monitor()
	}()

	f, _ := os.OpenFile(cfg.Address, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: f}))
	e.Logger.Fatal(e.Start(":8080"))
}

func (a API) Mapping(c echo.Context) error {
	var newMap request.Map

	if err := c.Bind(&newMap); err != nil {
		return err
	}

	if !newMap.Validate() {
		return c.String(http.StatusBadRequest, "This is not a url at all")
	}

	if newMap.ExpirationTime.Before(time.Now()) {
		var duration time.Duration = 5
		newMap.ExpirationTime = time.Now().Add(duration * time.Minute)
	}

	if newMap.ShortURL == "" {
		newMap = a.randomShortURL(newMap)
	} else if !a.customShortURL(newMap) {
		return c.String(http.StatusConflict, "This short url exists")
	}

	return c.JSON(http.StatusCreated, newMap)
}

func (a API) randomShortURL(new request.Map) request.Map {
	for {
		u := a.ShortURL.Choose()
		log.Print(u)
		new.ShortURL = u

		if err := a.Map.Insert(new.Model()); err == nil {
			return new
		}
	}
}

func (a API) customShortURL(newMap request.Map) bool {
	if err := a.Map.Insert(newMap.Model()); err != nil {
		return false
	}

	return true
}

func (a API) Redirect(c echo.Context) error {
	shortURL := c.Param("shortURL")
	if !a.CheckShortURL(shortURL) {
		return c.String(http.StatusBadRequest, shortURL)
	}

	mapping, err := a.Map.Retrieve(shortURL)

	if err != nil {
		return c.String(http.StatusNotFound, shortURL)
	}

	return c.String(http.StatusFound, mapping.LongURL)
}

func (a API) CheckShortURL(shortURL string) bool {
	fmt.Println("short url length")
	fmt.Println(len(shortURL))
	fmt.Println("cpnfig length")
	fmt.Println(a.URL.Length)
	//check the length of shortURL
	if len(shortURL) != a.URL.Length {
		return false
	}

	match, _ := regexp.MatchString("^[a-zA-Z]+$", shortURL)

	return match
}
