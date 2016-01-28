package ssemux

import (
	"errors"
	"sync"
)

const CLOSED_CH_BUFFER = 16

type SyncStream struct {
	key        string
	multiAssoc map[*MultiAssoc]string
	stream     *Stream
	rwmu       *sync.RWMutex
}

func NewSyncStream(key string, stream *Stream) *SyncStream {
	return &SyncStream{
		key:        key,
		multiAssoc: make(map[*MultiAssoc]string),
		stream:     stream,
		rwmu:       &sync.RWMutex{},
	}
}

func (ss *SyncStream) Associate(multiAssoc *MultiAssoc, assocKey string) {
	ss.multiAssoc[multiAssoc] = assocKey
}

func (ss *SyncStream) Disassociate() {
	for multiAssoc, assocKey := range ss.multiAssoc {
		multiAssoc.Disassociate(ss.key, assocKey)
	}
}

func (ss *SyncStream) Event(event *Event) {
	ss.rwmu.RLock()
	ss.stream.EventCh <- event
	ss.rwmu.RUnlock()
}

func (ss *SyncStream) Reset(stream *Stream) {
	ss.rwmu.Lock()
	ss.stream.ResetCh <- true
	ss.stream = stream
	ss.rwmu.Unlock()
}

type MultiAssoc struct {
	syncStream map[string]map[string]*SyncStream
	rwmu       *sync.RWMutex
}

func NewMultiAssoc() *MultiAssoc {
	return &MultiAssoc{
		syncStream: make(map[string]map[string]*SyncStream),
		rwmu:       &sync.RWMutex{},
	}
}

func (ma *MultiAssoc) Event(assocKey string, event *Event) {
	ma.rwmu.RLock()
	assoc, exists := ma.syncStream[assocKey]
	if exists{
		for _, v := range assoc {
			v.Event(event)
		}
	}
	ma.rwmu.RUnlock()
}

func (ma *MultiAssoc) Associate(key string, assocKey string, syncStream *SyncStream) {
	ma.rwmu.Lock()
	assoc, exists := ma.syncStream[assocKey]
	if !exists{
		assoc = make(map[string]*SyncStream)
		ma.syncStream[assocKey] = assoc
	}
	assoc[key] = syncStream
	ma.rwmu.Unlock()
}

func (ma *MultiAssoc) Disassociate(key string, assocKey string) {
	ma.rwmu.Lock()
	_, exists := ma.syncStream[assocKey]
	if exists{
		delete(ma.syncStream[assocKey], key)
	}
	ma.rwmu.Unlock()
}

type Store struct {
	closedCh     chan string
	multiAssoc   map[string]*MultiAssoc
	multiAssocMu *sync.Mutex
	resetMu      *sync.Mutex
	syncStream   map[string]*SyncStream
}

func NewStore() *Store {
	return &Store{
		closedCh:     make(chan string, CLOSED_CH_BUFFER),
		multiAssoc:   make(map[string]*MultiAssoc),
		multiAssocMu: &sync.Mutex{},
		resetMu:      &sync.Mutex{},
		syncStream:   make(map[string]*SyncStream),
	}
}

func (s *Store) Associate(key string, assoc string, assocKey string) error {
	syncStream, exists := s.syncStream[key]
	if !exists {
		return errors.New("key for stream does not exist")
	}
	ma, exists := s.multiAssoc[assoc]
	if !exists {
		s.multiAssocMu.Lock()
		ma, exists = s.multiAssoc[assoc]
		if !exists {
			ma = NewMultiAssoc()
			s.multiAssoc[assoc] = ma
		}
		s.multiAssocMu.Unlock()
	}
	ma.Associate(key, assocKey, syncStream)
	return nil
}

func (s *Store) New(key string) *Stream {
	s.resetMu.Lock()
	syncStream, exists := s.syncStream[key]
	stream := NewStream(key, s.closedCh)
	if exists {
		syncStream.Reset(stream)
	} else {
		s.syncStream[key] = NewSyncStream(key, stream)
	}
	s.resetMu.Unlock()
	return stream
}

func (s *Store) Event(assoc string, assocKey string, event *Event){
	ma, exists := s.multiAssoc[assoc]
	if (exists){
		ma.Event(assocKey, event)
	}
}