package main

import (
	"crypto/rand"
	"github.com/caleblloyd/ssemux-server/ssemux"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/net/http2"
	"log"
	"math/big"
	"net/http"
	"time"
	"strconv"
	"fmt"
)

const CONTEXT_KEY = 0
const SESSION_NAME = "sse"

func SessionHandler(cs *sessions.CookieStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := cs.Get(r, SESSION_NAME)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if session.IsNew {
			randId, _ := rand.Int(rand.Reader, big.NewInt(100000))
			randInt, _ := rand.Int(rand.Reader, big.NewInt(3))
			session.Values["id"] = randId.String()
			session.Values["uid"] = randInt.String()
		}
		session.Save(r, w)
		next.ServeHTTP(w, r)
	})
}

func StreamGenHandler(cs *sessions.CookieStore, ss *ssemux.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.
		session, err := cs.Get(r, SESSION_NAME)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Println(session.Values["id"].(string))
		fmt.Println(session.Values["uid"].(string))
		context.Set(r, CONTEXT_KEY, ss.New(session.ID))
		ss.Associate(session.ID, "uid", session.Values["uid"].(string))
		next.ServeHTTP(w, r)
	})
}

func main() {

	cs := sessions.NewCookieStore([]byte("something-very-secret"))
	ss := ssemux.NewStore()

	go func(){
		for {
			<- time.After(time.Second)
			for i:=int64(0); i<3; i++{
				iStr := strconv.FormatInt(i, 10)
				ss.Event("uid", iStr, &ssemux.Event{
					Comment: "test",
					Event: "yo",
					Data: "hi from "+iStr,
				})
			}
		}

	}()

	r := mux.NewRouter()
	r.Handle("/sse", SessionHandler(cs, StreamGenHandler(cs, ss, ssemux.Handler(CONTEXT_KEY))))
	s := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	http2.ConfigureServer(s, &http2.Server{})
	log.Fatal(s.ListenAndServe())
}
