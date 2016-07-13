package ssemux

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const EVENT_CH_BUFFER = 16
const HEARTBEAT_INTERVAL = time.Second * 15
const CLOSE_EVENT_FLAG = 0x1

type Event struct {
	// SSE fields
	Comment string
	Id      string
	Event   string
	Data    string
	// sender source to exclude from dissemination
	Source string
	// internal flags
	Flag int
}

func NewCloseEvent() *Event {
	return &Event{
		Event: "close",
		Data:  "true",
		Flag:  CLOSE_EVENT_FLAG,
	}
}

type Stream struct {
	key      string
	closedCh chan string
	EventCh  chan *Event
	ResetCh  chan bool
}

func NewStream(key string, closedCh chan string) *Stream {
	return &Stream{
		key:      key,
		closedCh: closedCh,
		EventCh:  make(chan *Event, EVENT_CH_BUFFER),
		ResetCh:  make(chan bool, 1),
	}
}

func (s *Stream) Handle(w http.ResponseWriter) {
	closeNotify := w.(http.CloseNotifier).CloseNotify()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	//		fmt.Fprintf(w, ": ~2KB of junk to force browsers to start rendering immediately:\r\n")
	//		io.WriteString(w, strings.Repeat(": xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\r\n", 32))
	//		w.(http.Flusher).Flush()

	var b []byte
	var e *Event
	for {
		select {
		case <-closeNotify:
			s.closedCh <- s.key
			return
		case e = <-s.EventCh:
			fmt.Println(e)
			b = make([]byte, 0)
			if e.Comment != "" {
				b = append(b, prefixLines(":", strings.NewReader(e.Comment))...)
			}
			if e.Id != "" {
				b = append(b, []byte("id:"+e.Id+"\r\n")...)
			}
			if e.Event != "" {
				b = append(b, []byte("event:"+e.Event+"\r\n")...)
			}
			if e.Data != "" {
				b = append(b, prefixLines("data:", strings.NewReader(e.Data))...)
			}
			b = append(b, []byte("\r\n")...)
			w.Write(b)
			w.(http.Flusher).Flush()
			if e.Flag&CLOSE_EVENT_FLAG == CLOSE_EVENT_FLAG {
				return
			}
		case <-s.ResetCh:
			s.EventCh <- NewCloseEvent()
		case <-time.After(HEARTBEAT_INTERVAL):
			s.EventCh <- &Event{
				Comment: "heartbeat",
			}
		}

	}
}
