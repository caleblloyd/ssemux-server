package main

import (
	"crypto/rand"
	"github.com/caleblloyd/ssemux-server/ssemux"
	"github.com/go-ozzo/ozzo-routing"
	//	"github.com/go-ozzo/ozzo-routing/access"
	//	"github.com/go-ozzo/ozzo-routing/slash"
	//	"github.com/go-ozzo/ozzo-routing/fault"
	"github.com/gorilla/sessions"
	//	"golang.org/x/net/http2"
	//	"log"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

const STREAM_CONTEXT_KEY = "ssemux-stream"
const SESSION_NAME = "sse"

func SessionMiddleware(cs *sessions.CookieStore) routing.Handler {
	return func(c *routing.Context) error {
		session, err := cs.Get(c.Request, SESSION_NAME)
		if err != nil {
			return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if session.IsNew {
			randId, _ := rand.Int(rand.Reader, big.NewInt(100000))
			randInt, _ := rand.Int(rand.Reader, big.NewInt(3))
			session.Values["id"] = randId.String()
			session.Values["uid"] = randInt.String()
		}
		session.Save(c.Request, c.Response)
		return nil
	}
}

func StreamCreateMiddleware(cs *sessions.CookieStore, ss *ssemux.Store) routing.Handler {
	return func(c *routing.Context) error {
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.
		session, err := cs.Get(c.Request, SESSION_NAME)
		if err != nil {
			return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		fmt.Println("id:  " + session.Values["id"].(string))
		fmt.Println("uid: " + session.Values["uid"].(string))
		c.Set(STREAM_CONTEXT_KEY, ss.New(session.Values["id"].(string)))
		ss.Associate(session.Values["id"].(string), "uid", session.Values["uid"].(string))
		return nil
	}
}

func main() {

	cs := sessions.NewCookieStore([]byte("something-very-secret"))
	ss := ssemux.NewStore()

	go func() {
		for {
			<-time.After(time.Second)
			for i := int64(0); i < 3; i++ {
				iStr := strconv.FormatInt(i, 10)
				ss.Event("uid", iStr, &ssemux.Event{
					Comment: "test",
					Event:   "yo",
					Data:    "hi from " + iStr,
				})
			}
		}
	}()

	r := routing.New()
	//	r.Use(
	//		access.Logger(log.Printf),
	//		slash.Remover(http.StatusMovedPermanently),
	//		fault.Recovery(log.Printf),
	//
	//	)
	sse := r.Group("/sse")
	sse.Use(
		SessionMiddleware(cs),
		StreamCreateMiddleware(cs, ss),
	)
	sse.Get("/test", func(c *routing.Context) error {
		s := c.Get(STREAM_CONTEXT_KEY).(*ssemux.Stream)
		if s == nil {
			return routing.NewHTTPError(http.StatusInternalServerError, "could not find stream in context")
		}
		s.Handle(c.Response)
		return nil
	})
	//	s := &http.Server{
	//		Addr:    ":8080",
	//		Handler: r,
	//	}
	//	http2.ConfigureServer(s, &http2.Server{})
	//	log.Fatal(s.ListenAndServe())
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
