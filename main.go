package main

import (
	"github.com/ismdeep/insight-hub-rss/api"
	"github.com/ismdeep/insight-hub-rss/syncer"
)

func main() {
	go syncer.Run()
	api.Run()
}
