package core

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Link        string    `json:"link"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Content     string    `json:"content"`
	PublishedAt time.Time `json:"published_at"`
}

func LinkToRecordID(link string) string {
	s := sha256.New()
	s.Write([]byte(link))
	return fmt.Sprintf("%x", s.Sum(nil))
}

func RecordID(r Record) string {
	return LinkToRecordID(r.Link)
}

func RecordMarshal(r Record) string {
	id := RecordID(r)
	line := fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v",
		id,
		r.PublishedAt.UnixNano(),
		url.QueryEscape(r.Source),
		url.QueryEscape(r.Link),
		url.QueryEscape(r.Title),
		url.QueryEscape(r.Author),
		url.QueryEscape(r.Content))
	return line
}

func RecordUnmarshal(line string) (Record, error) {
	items := strings.Split(line, "|")
	if len(items) != 7 {
		return Record{}, errors.New("invalid line")
	}

	// id := items[0]
	publishedAtStr := items[1]
	source, err := url.QueryUnescape(items[2])
	if err != nil {
		return Record{}, err
	}

	link, err := url.QueryUnescape(items[3])
	if err != nil {
		return Record{}, err
	}

	title, err := url.QueryUnescape(items[4])
	if err != nil {
		return Record{}, err
	}

	author, err := url.QueryUnescape(items[5])
	if err != nil {
		return Record{}, err
	}

	content, err := url.QueryUnescape(items[6])
	if err != nil {
		return Record{}, err
	}

	pNano, err := strconv.ParseInt(publishedAtStr, 10, 64)
	if err != nil {
		return Record{}, err
	}

	return Record{
		Source:      source,
		Link:        link,
		Title:       title,
		Author:      author,
		Content:     content,
		PublishedAt: time.Unix(pNano/1_000_000_000, pNano%1_000_000_000),
	}, nil
}
