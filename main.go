package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	letters     = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	secretToken = "verysecrettoken2"
	calanderId  = "gerardyap@17.media"
)

type handler struct {
	Service *calendar.Service
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	b, err := ioutil.ReadFile("privateKey.pem")
	if err != nil {
		log.Fatalf("Unable to read private key file: %v", err)
	}

	jwtConf := &jwt.Config{
		TokenURL:   google.JWTTokenURL,
		Scopes:     []string{calendar.CalendarReadonlyScope},
		PrivateKey: b,
		Email:      "yap-calander@live17-sre.iam.gserviceaccount.com",
	}

	calanderService, err := calendar.NewService(ctx, option.WithTokenSource(jwtConf.TokenSource(ctx)))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	ch := calendar.Channel{
		Address:    "https://aed2-111-241-102-91.ngrok.io",
		Type:       "web_hook",
		Id:         randSeq(10),
		Token:      secretToken,
		Expiration: time.Now().Add(3600 * time.Hour).UnixNano(),
	}
	chanResp, err := calanderService.Events.Watch("gerardyap@17.media", &ch).Do()
	if err != nil {
		log.Fatalf("Unable to watch event: %v", err)
	}
	ch.ResourceId = chanResp.ResourceId

	server := &http.Server{
		Addr: ":8080",
	}

	// graceful shutdown
	go func() {
		for sig := range sigChan {
			fmt.Printf("Receiving signal %v\n", sig)
			fmt.Printf("Stopping channal %s\n", ch.ResourceId)
			err := calanderService.Channels.Stop(&ch).Do()
			if err != nil {
				fmt.Printf("Error stopping channel: %v\n", err)
			}
			server.Shutdown(ctx)
		}
	}()

	customHandler := &handler{
		Service: calanderService,
	}
	http.Handle("/", customHandler)
	server.ListenAndServe()
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Goog-Channel-Token")
	if token != secretToken {
		fmt.Println("forbidden")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
		return
	}

	sync := r.Header.Get("X-Goog-Resource-State")
	if sync == "sync" {
		fmt.Println("sync")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	// print header
	for k, v := range r.Header {
		fmt.Println(k, v)
	}

	// print body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
