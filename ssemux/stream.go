package ssemux

import(
	"github.com/gorilla/context"
	"net/http"
	"time"
	"io"
	"strings"
)

const EVENT_CH_BUFFER = 16
const HEARTBEAT_INTERVAL = time.Second * 15

type Event struct{
	Comment       string
	CommentReader io.Reader
	Event 		  string
	Data    	  string
	DataReader	  io.Reader
}

type Stream struct{
	key       string
	closedCh  chan string
	EventCh   chan *Event
	ResetCh   chan bool

}

func NewStream(key string, closedCh chan string) *Stream {
	return &Stream{
		key:      key,
		closedCh: closedCh,
		EventCh:  make(chan *Event, EVENT_CH_BUFFER),
		ResetCh:  make(chan bool, 1),
	}
}

func Handler(contextKey int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs, found := context.GetOk(r, contextKey)
		if !found {
			http.Error(w, "Cannot get context for event stream", 500)
			return
		}
		s := fs.(*Stream)
		closeNotify := w.(http.CloseNotifier).CloseNotify()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()

//		fmt.Fprintf(w, ": ~2KB of junk to force browsers to start rendering immediately:\r\n")
//		io.WriteString(w, strings.Repeat(": xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\r\n", 32))
//		w.(http.Flusher).Flush()

		var b []byte
		var e *Event
		for{
			select{
			case <- closeNotify:
				s.closedCh <- s.key
				return
			case e = <- s.EventCh:
				b = make([]byte, 0)
				if e.CommentReader == nil && e.Comment != ""{
					e.CommentReader = strings.NewReader(e.Comment)
				}
				if e.CommentReader != nil{
					b = append(b, prefixLines(": ", e.CommentReader)...)
				}
				if e.Event != ""{
					b = append(b, []byte("event: " + e.Event + "\r\n")...)
				}
				if e.DataReader == nil && e.Data != ""{
					e.DataReader = strings.NewReader(e.Data)
				}
				if e.DataReader != nil{
					b = append(b, prefixLines(": ", e.DataReader)...)
				}
				b = append(b, []byte("\r\n")...)
				w.Write(b)
				w.(http.Flusher).Flush()
			case <- s.ResetCh:
				return
			case <- time.After(HEARTBEAT_INTERVAL):
				s.EventCh <- &Event{
					Comment: "heartbeat",
				}
			}
		}
	})
}
