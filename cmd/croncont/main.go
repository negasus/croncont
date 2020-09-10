package main

import (
	"bytes"
	"context"
	"github.com/cristalhq/aconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	version = "undefined"
)

type config struct {
	URL            string `default:"http://localhost" usage:"request url"`
	Method         string `default:"POST" usage:"request http method"`
	Body           string `default:"" usage:"request body"`
	Headers        string `default:"" usage:"request headers. Example: Authorization=Foo|X-Foo=Baz"`
	Timeout        int    `default:"3000" usage:"request timeout, ms"`
	ExpectedStatus int    `default:"200" usage:"expected response status code. 0 for no check"`

	Listen string `default:"0.0.0.0:8001" usage:"listen address for healthcheck and metrics, disable if empty"`

	Spec    string `default:"0 * * * * *" usage:"cron spec"`
	Verbose bool   `default:"false" usage:"verbose mode"`
}

func main() {
	log.Printf("croncont version %s", version)

	var (
		calls = promauto.NewCounter(prometheus.CounterOpts{
			Name: "croncont_calls",
			Help: "The total number of calls",
		})
	)

	loader := aconfig.LoaderFor(&config{}).
		WithEnvPrefix("CRON").
		Build()

	var cfg config
	if err := loader.Load(&cfg); err != nil {
		log.Printf("error load configuration, %v", err)
		os.Exit(1)
	}

	if cfg.Verbose {
		log.Printf("verbose mode on")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	if cfg.Listen != "" {
		go func() {
			if err := listen(ctx, ctxCancel, cfg.Listen); err != nil {
				log.Printf("error listen, %v", err)
			}
		}()
	}
	defer ctxCancel()

	req, err := http.NewRequest(cfg.Method, cfg.URL, bytes.NewReader([]byte(cfg.Body)))
	if err != nil {
		log.Printf("error create request, %v", err)
		return
	}
	if len(cfg.Headers) > 0 {
		for _, h := range strings.Split(cfg.Headers, "|") {
			h = strings.TrimSpace(h)
			if len(h) == 0 {
				continue
			}
			v := strings.Split(h, "=")
			if len(v) != 2 {
				log.Printf("error headers format")
				os.Exit(1)
			}
			req.Header.Add(v[0], v[1])
		}
	}

	httpClient := &http.Client{
		Timeout: time.Millisecond * time.Duration(cfg.Timeout),
	}

	c := cron.New(cron.WithSeconds())

	entryID, err := c.AddFunc(cfg.Spec, func() {
		calls.Inc()
		if cfg.Verbose {
			log.Printf("Call %s %s", cfg.Method, cfg.URL)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("error send request, %v", err)
			return
		}
		resp.Body.Close()
		if cfg.ExpectedStatus == 0 {
			return
		}

		if resp.StatusCode != cfg.ExpectedStatus {
			log.Printf("unexpected response statuc code %d", resp.StatusCode)
		}
	})

	if err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}

	_ = entryID

	log.Printf("start with config %#v", cfg)

	c.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	signal.Notify(ch, syscall.SIGINT)

	select {
	case <-ch:
		ctxCancel()
	case <-ctx.Done():
	}

	c.Stop()

	log.Printf("done")
}

func listen(ctx context.Context, ctxCancel context.CancelFunc, address string) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", livenessHandler)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		log.Printf("listen %s", ln.Addr().String())
		if err := server.Serve(ln); err != nil {
			log.Printf("error serve, %v", err)
			ctxCancel()
		}
	}()

	<-ctx.Done()
	return server.Shutdown(ctx)
}

func livenessHandler(_ http.ResponseWriter, _ *http.Request) {}
