package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/insight-hub-rss/conf"
	"github.com/ismdeep/insight-hub-rss/pkg/core"
	"github.com/ismdeep/insight-hub-rss/store"
)

type Worker struct {
	link              string
	lastContentLength string
	lastEtag          string
}

func NewWorker(link string) *Worker {
	return &Worker{
		link: link,
	}
}

func (receiver *Worker) work(ctx context.Context) {
	for {
		log.WithContext(ctx).Info("start", zap.String("link", receiver.link))
		if err := receiver.check(); err != nil {
			log.WithContext(ctx).Info("start", zap.String("failed", receiver.link))
			time.Sleep(10 * time.Second)
			continue
		}
		log.WithContext(ctx).Info("completed", zap.String("link", receiver.link))
		time.Sleep(10 * time.Minute)
	}
}

func (receiver *Worker) DownloadByID(id string, source string, contentLink string) error {
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(contentLink)
	if err != nil {
		return fmt.Errorf("failed to download content: %v\n", err)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read content: %v\n", err)
	}

	var r core.Record
	if err := json.Unmarshal(content, &r); err != nil {
		return fmt.Errorf("failed to unmarshal content: %v\n", err)
	}

	if r.ID != id {
		return fmt.Errorf("content link check failed: id is not correct")
	}

	if r.Source != source {
		return fmt.Errorf("content link check failed: source is not correct")
	}

	if err := store.RecordSave(store.Record{
		ID:          id,
		Source:      r.Source,
		Author:      r.Author,
		Title:       r.Title,
		Link:        r.Link,
		Content:     r.Content,
		PublishedAt: r.PublishedAt.UnixNano(),
	}); err != nil {
		return fmt.Errorf("failed to save record: %v", err.Error())
	}

	return nil
}

func (receiver *Worker) check() error {
	headResp, err := (&http.Client{Timeout: 30 * time.Second}).Head(receiver.link)
	if err != nil {
		return err
	}

	if headResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to request via head method: %v [%v]", receiver.link, headResp.Status)
	}

	etag := headResp.Header.Get("Etag")
	contentLength := headResp.Header.Get("Content-Length")
	if receiver.lastEtag == etag && receiver.lastContentLength == contentLength {
		fmt.Printf("[INFO] skip to get content due etag and content-length not changed.")
		return nil
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(receiver.link)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	lineChan := make(chan string, 1024)
	go func() {
		for _, line := range strings.Split(string(raw), "\n") {
			lineChan <- line
		}
		close(lineChan)
	}()

	errChan := make(chan error, 1024)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := log.NewTraceContext(uuid.NewString())

			for line := range lineChan {

				if line == "" {
					// skip if line is empty
					continue
				}

				log.WithContext(ctx).Info("process", zap.String("line", line))

				items := strings.Split(line, "|")
				if len(items) != 2 {
					// skip if line format is invalid
					log.WithContext(ctx).Warn("skip due line format is invalid")
					continue
				}

				id := items[0]
				source := items[1]

				if id == "" || source == "" {
					// skip if id or source is empty
					log.WithContext(ctx).Warn("skip due id or source is empty")
					continue
				}

				contentLink := fmt.Sprintf("%v/%v.d/%v.json",
					receiver.link[:strings.LastIndex(receiver.link, "/")],
					source,
					id)
				if err := receiver.DownloadByID(id, source, contentLink); err != nil {
					log.WithContext(ctx).Error("failed to download by id", zap.Error(err))
					errChan <- fmt.Errorf("failed to download by id: %v", err.Error())
					continue
				}
			}

		}()
	}

	var errLst []error
	var wgErr sync.WaitGroup
	wgErr.Add(1)
	go func() {
		defer wgErr.Done()
		for err := range errChan {
			errLst = append(errLst, err)
		}
	}()

	wg.Wait()
	close(errChan)

	wgErr.Wait()

	if len(errLst) > 0 {
		return errors.Join(errLst...)
	}

	// update etag and content-length
	receiver.lastEtag = etag
	receiver.lastContentLength = contentLength
	return nil
}

func Run() {
	var wg sync.WaitGroup
	for _, link := range conf.Links {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			NewWorker(link).work(log.NewTraceContext(uuid.NewString()))
		}(link)
	}
	wg.Wait()
}
