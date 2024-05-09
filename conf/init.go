package conf

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed default-links.txt
var defaultLinks string

var Links []string

func init() {
	// Load env
	if os.Getenv("INSIGHT_HUB_LINKS") == "" {
		if err := os.Setenv("INSIGHT_HUB_LINKS", defaultLinks); err != nil {
			panic(fmt.Errorf("failed to sec env: INSIGHT_HUB_LINKS, err: %v", err.Error()))
		}
	}

	links := strings.Split(os.Getenv("INSIGHT_HUB_LINKS"), "\n")
	for _, link := range links {
		link = strings.TrimSpace(link)
		if link != "" {
			Links = append(Links, link)
		}
	}
}
