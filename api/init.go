package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"

	"github.com/ismdeep/insight-hub-rss/store"
)

//go:embed index.txt
var indexText string

//go:embed favicon.png
var favicon []byte

var eng *gin.Engine

func init() {
	eng = gin.Default()

	eng.GET("/", func(c *gin.Context) {
		sources, err := store.RecordSources()
		if err != nil {
			fmt.Println(err)
		}
		var content strings.Builder
		for _, source := range sources {
			content.WriteString(fmt.Sprintf("https://insight-hub.ismdeep.com/%v/rss for %v RSS Feed.\n", source, source))
		}
		c.String(http.StatusOK, indexText+content.String())
	})

	eng.GET("/favicon.ico", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "image/png")
		c.Writer.Header().Set("Expires", time.Now().Add(24*time.Hour).UTC().Format(http.TimeFormat))
		_, _ = c.Writer.Write(favicon)
	})

	eng.GET("/rss", func(c *gin.Context) {
		lst, err := store.RecordRecentList()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		feed := &feeds.Feed{
			Title:       "All - InsightHub RSS Feed",
			Link:        &feeds.Link{Href: "https://insight-hub.ismdeep.com/rss"},
			Description: "A collection of RSS feeds.",
			Author:      &feeds.Author{Name: "L. Jiang", Email: "l.jiang.1024@gmail.com"},
			Created:     time.Now(),
		}

		for _, item := range lst {
			feed.Items = append(feed.Items, &feeds.Item{
				Title:     item.Title,
				Link:      &feeds.Link{Href: item.Link},
				Source:    &feeds.Link{Href: item.Link},
				Author:    &feeds.Author{Name: item.Author, Email: ""},
				Id:        item.ID,
				Updated:   time.Unix(item.PublishedAt/1_000_000_000, item.PublishedAt%1_000_000_000),
				Created:   time.Unix(item.PublishedAt/1_000_000_000, item.PublishedAt%1_000_000_000),
				Enclosure: nil,
				Content:   item.Content,
			})
		}

		// Render the feed as XML
		rss, err := feed.ToRss()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to generate RSS")
			return
		}

		// Set the response content type to XML
		c.Data(http.StatusOK, "application/rss+xml", []byte(rss))
	})

	eng.GET("/:source/rss", func(c *gin.Context) {
		source := c.Param("source")

		metaInfo, err := store.MetaInfo(source)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		lst, err := store.RecordRecentListBySource(source)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		feed := &feeds.Feed{
			Title:       fmt.Sprintf("%v - InsightHub", metaInfo.Name),
			Link:        &feeds.Link{Href: fmt.Sprintf("https://insight-hub.ismdeep.com/%v/rss", source)},
			Description: "A collection of RSS feeds.",
			Author:      &feeds.Author{Name: "L. Jiang", Email: "l.jiang.1024@gmail.com"},
			Created:     time.Now(),
		}

		for _, item := range lst {
			feed.Items = append(feed.Items, &feeds.Item{
				Title:     item.Title,
				Link:      &feeds.Link{Href: item.Link},
				Source:    &feeds.Link{Href: item.Link},
				Author:    &feeds.Author{Name: item.Author, Email: ""},
				Id:        item.ID,
				Updated:   time.Unix(item.PublishedAt/1_000_000_000, item.PublishedAt%1_000_000_000),
				Created:   time.Unix(item.PublishedAt/1_000_000_000, item.PublishedAt%1_000_000_000),
				Enclosure: nil,
				Content:   item.Content,
			})
		}

		// Render the feed as XML
		rss, err := feed.ToRss()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to generate RSS")
			return
		}

		// Set the response content type to XML
		c.Data(http.StatusOK, "application/rss+xml", []byte(rss))
	})

}

func Run() {
	if err := eng.Run(":8080"); err != nil {
		panic(err)
	}
}
