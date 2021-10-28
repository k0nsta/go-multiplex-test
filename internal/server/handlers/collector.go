package handlers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/k0nsta/go-multiplex-test/internal/semaphore"
)

const (
	notAllowedMethodErrMsg = "allowed only POST method"
	requestLimitErrMsg     = "exceeded simultaneous request limit"
	urlLimitErrMsg         = "exceeded maximum URLs per request"
	outcomeErrMsg          = "failed to perform outcome requests"
)

// type Response struct {
// 	Payload []Payload `json:"payload,omitempty"`
// }

type Payload struct {
	URL     string `json:"url"`
	Payload string `json:"payload"`
}

type Collector struct {
	OutConnPool int
	MaxURLs     int
	outTimeout  time.Duration
	reqPool     *semaphore.Semaphore
}

// NewCollector constructs collector handler.
func NewCollector(maxReq, maxOutConn, maxURLs uint, timeout time.Duration) *Collector {
	log.Printf("[INFO][%s] request limit: %d, outgoing limit: %d", time.Now().Format(time.RFC3339), maxReq, maxOutConn)

	return &Collector{
		OutConnPool: int(maxOutConn),
		MaxURLs:     int(maxURLs),
		outTimeout:  timeout,
		reqPool:     semaphore.New(maxReq, time.Second*time.Duration(1)),
	}
}

//nolint:stylecheck
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		c.post(w, r)
	default:
		c.errorHandler(w, notAllowedMethodErrMsg, http.StatusMethodNotAllowed)
	}
}

func (c *Collector) post(w http.ResponseWriter, r *http.Request) {
	var urls []string

	if err := c.reqPool.Acquire(); err != nil {
		c.errorHandler(w, requestLimitErrMsg, http.StatusTooManyRequests)

		return
	}

	defer c.reqPool.Release()

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&urls)
	if err != nil {
		c.errorHandler(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if len(urls) > c.MaxURLs {
		c.errorHandler(w, urlLimitErrMsg, http.StatusBadRequest)

		return
	}

	respPayload, err := c.processRequests(r.Context(), urls)
	if err != nil {
		c.errorHandler(w, outcomeErrMsg, http.StatusGatewayTimeout)

		return
	}

	payload, err := json.Marshal(respPayload)
	if err != nil {
		c.errorHandler(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Write(payload) //nolint:errcheck
}

//nolint:funlen
func (c *Collector) processRequests(ctx context.Context, urls []string) ([]Payload, error) {
	rCtx, outCtxCancel := context.WithCancel(ctx)
	defer outCtxCancel()

	client := new(http.Client)

	inChan := make(chan string, len(urls))
	outChan := make(chan Payload, len(urls))

	var wg sync.WaitGroup
	for i := 0; i < c.OutConnPool; i++ {
		wg.Add(1)
		go func(ctx context.Context, tasksChan chan string, result chan Payload) {
			defer wg.Done()

			for url := range tasksChan {
				select {
				case <-ctx.Done():
					log.Printf("[DEBUG][%s] close contex %s", time.Now().Format(time.RFC3339), ctx.Err())

					return

				default:
					response, err := request(rCtx, url, client, c.outTimeout)
					if err != nil {
						log.Printf("[ERROR][%s] failed perform request: %s", time.Now().Format(time.RFC3339), err)
						outCtxCancel()

						return
					}

					rp := Payload{URL: url, Payload: string(response)}
					outChan <- rp
				}
			}
		}(rCtx, inChan, outChan)
	}

	for _, url := range urls {
		inChan <- url
	}
	close(inChan)

	wg.Wait()
	close(outChan)

	var respPload []Payload

	select {
	case <-rCtx.Done():
		return respPload, rCtx.Err()
	default:
	}

	for r := range outChan {
		respPload = append(respPload, r)
	}

	return respPload, nil
}

func (c *Collector) errorHandler(w http.ResponseWriter, errMsg string, errCode int) {
	w.WriteHeader(errCode)

	log.Printf("[ERROR][%s]: %s", time.Now().Format(time.RFC3339), errMsg)
	if errCode != 500 {
		w.Write([]byte(errMsg)) //nolint:errcheck
	}
}

func request(ctx context.Context, url string, client *http.Client, timeout time.Duration) ([]byte, error) {
	var data []byte

	timeCtx, timeCtxCancel := context.WithTimeout(ctx, timeout)
	defer timeCtxCancel()

	req, err := http.NewRequestWithContext(timeCtx, "GET", url, nil)
	if err != nil {
		return data, err
	}

	res, err := client.Do(req.WithContext(timeCtx))
	if err != nil {
		return data, err
	}
	defer res.Body.Close()

	data, err = ioutil.ReadAll(res.Body)

	return data, err
}
