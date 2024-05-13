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
	metaLink                   string
	metaInfo                   MetaInfo
	indexLink                  string
	contentLinkPrefix          string
	indexLinkLastContentLength string
	indexLinkLastEtag          string
}

func NewWorker(metaLink string) (*Worker, error) {
	metaInfo, err := DownloadMeta(metaLink)
	if err != nil {
		log.WithContext(context.Background()).Error("failed to download meta info", zap.Error(err))
		return nil, err
	}

	indexPrefix := metaLink[:strings.LastIndex(metaLink, "/")]

	if err := store.MetaSave(&store.Meta{
		Source:   metaInfo.Source,
		Name:     metaInfo.Name,
		HomePage: metaInfo.HomePage,
	}); err != nil {
		log.WithContext(context.Background()).Error("failed to save meta info", zap.Error(err))
		return nil, err
	}

	return &Worker{
		metaLink:                   metaLink,
		metaInfo:                   *metaInfo,
		indexLink:                  fmt.Sprintf("%v/%v.txt", indexPrefix, metaInfo.Source),
		contentLinkPrefix:          fmt.Sprintf("%v/%v.d", indexPrefix, metaInfo.Source),
		indexLinkLastContentLength: "",
		indexLinkLastEtag:          "",
	}, nil
}

func (receiver *Worker) work(ctx context.Context) {
	for {
		log.WithContext(ctx).Info("start", zap.String("link", receiver.indexLink))
		if err := receiver.check(); err != nil {
			log.WithContext(ctx).Info("start", zap.String("failed", receiver.indexLink))
			time.Sleep(10 * time.Second)
			continue
		}
		log.WithContext(ctx).Info("completed", zap.String("link", receiver.indexLink))
		time.Sleep(10 * time.Minute)
	}
}

type MetaInfo struct {
	Source   string `json:"source"`
	HomePage string `json:"home_page"`
	Name     string `json:"name"`
}

func DownloadMeta(metaLink string) (*MetaInfo, error) {
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(metaLink)
	if err != nil {
		return nil, fmt.Errorf("failed to download meta file: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithContext(context.Background()).Warn("failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download meta file, status: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download meta file: %w", err)
	}

	var metaInfo MetaInfo
	if err := json.Unmarshal(content, &metaInfo); err != nil {
		return nil, fmt.Errorf("failed to download meta file: %w", err)
	}

	return &metaInfo, nil
}

func (receiver *Worker) DownloadContent(id string) error {
	contentLink := fmt.Sprintf("%v/%v.json", receiver.contentLinkPrefix, id)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(contentLink)
	if err != nil {
		return fmt.Errorf("failed to download content: %v", err)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read content: %v", err)
	}

	var r core.Record
	if err := json.Unmarshal(content, &r); err != nil {
		return fmt.Errorf("failed to unmarshal content: %v", err)
	}

	if r.ID != id {
		return fmt.Errorf("content link check failed: id is not correct")
	}

	if r.Source != receiver.metaInfo.Source {
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
	headResp, err := (&http.Client{Timeout: 30 * time.Second}).Head(receiver.indexLink)
	if err != nil {
		return err
	}

	if headResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to request via head method: %v [%v]", receiver.indexLink, headResp.Status)
	}

	etag := headResp.Header.Get("Etag")
	contentLength := headResp.Header.Get("Content-Length")
	if receiver.indexLinkLastEtag == etag && receiver.indexLinkLastContentLength == contentLength {
		fmt.Printf("[INFO] skip to get content due etag and content-length not changed.")
		return nil
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(receiver.indexLink)
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

				if err := receiver.DownloadContent(id); err != nil {
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
	receiver.indexLinkLastEtag = etag
	receiver.indexLinkLastContentLength = contentLength
	return nil
}

func Run() {
	var wg sync.WaitGroup
	for _, link := range conf.Links {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			w, err := NewWorker(link)
			if err != nil {
				log.WithContext(context.Background()).Fatal("failed to create worker", zap.String("link", link), zap.Error(err))
				return
			}
			w.work(log.NewTraceContext(uuid.NewString()))
		}(link)
	}
	wg.Wait()
}
