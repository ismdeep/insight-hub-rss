package main

import (
	"fmt"

	"github.com/ismdeep/insight-hub-rss/api"
	"github.com/ismdeep/insight-hub-rss/conf"
	"github.com/ismdeep/insight-hub-rss/syncer"
)

func main() {
	for _, link := range conf.Links {
		fmt.Println(link)
	}

	go syncer.Run()
	api.Run()
}
