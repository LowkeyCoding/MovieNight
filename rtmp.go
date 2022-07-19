package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/nareix/joy5/av"
	"github.com/nareix/joy5/format/flv"
	"github.com/nareix/joy5/format/rtmp"
	"github.com/zorchenhimer/MovieNight/common"
)

type gopCacheSnapshot struct {
	pkts []av.Packet
	idx  int
}

type gopCache struct {
	pkts  []av.Packet
	idx   int
	curst unsafe.Pointer
}

func (gc *gopCache) put(pkt av.Packet) {
	if pkt.IsKeyFrame {
		gc.pkts = []av.Packet{}
	}
	gc.pkts = append(gc.pkts, pkt)
	gc.idx++
	st := &gopCacheSnapshot{
		pkts: gc.pkts,
		idx:  gc.idx,
	}
	atomic.StorePointer(&gc.curst, unsafe.Pointer(st))
}

func (gc *gopCache) curSnapshot() *gopCacheSnapshot {
	return (*gopCacheSnapshot)(atomic.LoadPointer(&gc.curst))
}

type gopCacheReadCursor struct {
	lastidx int
}

func (rc *gopCacheReadCursor) advance(cur *gopCacheSnapshot) []av.Packet {
	lastidx := rc.lastidx
	rc.lastidx = cur.idx
	if diff := cur.idx - lastidx; diff <= len(cur.pkts) {
		return cur.pkts[len(cur.pkts)-diff:]
	} else {
		return cur.pkts
	}
}

type mergeSeqhdr struct {
	cb     func(av.Packet)
	hdrpkt av.Packet
}

func (m *mergeSeqhdr) do(pkt av.Packet) {
	switch pkt.Type {
	case av.H264DecoderConfig:
		m.hdrpkt.VSeqHdr = append([]byte(nil), pkt.Data...)
	case av.H264:
		pkt.Metadata = m.hdrpkt.Metadata
		if pkt.IsKeyFrame {
			pkt.VSeqHdr = m.hdrpkt.VSeqHdr
		}
		m.cb(pkt)
	case av.AACDecoderConfig:
		m.hdrpkt.ASeqHdr = append([]byte(nil), pkt.Data...)
	case av.AAC:
		pkt.Metadata = m.hdrpkt.Metadata
		pkt.ASeqHdr = m.hdrpkt.ASeqHdr
		m.cb(pkt)
	case av.Metadata:
		m.hdrpkt.Metadata = pkt.Data
	}
}

type splitSeqhdr struct {
	cb     func(av.Packet) error
	hdrpkt av.Packet
}

func (s *splitSeqhdr) sendmeta(pkt av.Packet) error {
	if bytes.Compare(s.hdrpkt.Metadata, pkt.Metadata) != 0 {
		if err := s.cb(av.Packet{
			Type: av.Metadata,
			Data: pkt.Metadata,
		}); err != nil {
			return err
		}
		s.hdrpkt.Metadata = pkt.Metadata
	}
	return nil
}

func (s *splitSeqhdr) do(pkt av.Packet) error {
	switch pkt.Type {
	case av.H264:
		if err := s.sendmeta(pkt); err != nil {
			return err
		}
		if pkt.IsKeyFrame {
			if bytes.Compare(s.hdrpkt.VSeqHdr, pkt.VSeqHdr) != 0 {
				if err := s.cb(av.Packet{
					Type: av.H264DecoderConfig,
					Data: pkt.VSeqHdr,
				}); err != nil {
					return err
				}
				s.hdrpkt.VSeqHdr = pkt.VSeqHdr
			}
		}
		return s.cb(pkt)
	case av.AAC:
		if err := s.sendmeta(pkt); err != nil {
			return err
		}
		if bytes.Compare(s.hdrpkt.ASeqHdr, pkt.ASeqHdr) != 0 {
			if err := s.cb(av.Packet{
				Type: av.AACDecoderConfig,
				Data: pkt.ASeqHdr,
			}); err != nil {
				return err
			}
			s.hdrpkt.ASeqHdr = pkt.ASeqHdr
		}
		return s.cb(pkt)
	}
	return nil
}

type streamSub struct {
	notify chan struct{}
}

type streamPub struct {
	cancel func()
	gc     *gopCache
}

type stream struct {
	n   int64
	sub sync.Map
	pub unsafe.Pointer
}

func (s *stream) curGopCacheSnapshot() *gopCacheSnapshot {
	sp := (*streamPub)(atomic.LoadPointer(&s.pub))
	if sp == nil {
		return nil
	}
	return sp.gc.curSnapshot()
}

func (s *stream) addSub(close <-chan bool, w av.PacketWriter) {
	ss := &streamSub{
		notify: make(chan struct{}, 1),
	}

	s.sub.Store(ss, nil)
	defer s.sub.Delete(ss)

	var cursor *gopCacheReadCursor
	var lastsp *streamPub

	seqsplit := splitSeqhdr{
		cb: func(pkt av.Packet) error {
			return w.WritePacket(pkt)
		},
	}

	for {
		var pkts []av.Packet

		sp := (*streamPub)(atomic.LoadPointer(&s.pub))
		if sp != lastsp {
			cursor = &gopCacheReadCursor{}
			lastsp = sp
		}
		if sp != nil {
			cur := sp.gc.curSnapshot()
			if cur != nil {
				pkts = cursor.advance(cur)
			}
		}

		if len(pkts) == 0 {
			select {
			case <-close:
				return
			case <-ss.notify:
			}
		} else {
			for _, pkt := range pkts {
				if err := seqsplit.do(pkt); err != nil {
					return
				}
			}
		}
	}
}

func (s *stream) notifySub() {
	s.sub.Range(func(key, value interface{}) bool {
		ss := key.(*streamSub)
		select {
		case ss.notify <- struct{}{}:
		default:
		}
		return true
	})
}

func (s *stream) setPub(r av.PacketReader) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sp := &streamPub{
		cancel: cancel,
		gc:     &gopCache{},
	}

	oldsp := (*streamPub)(atomic.SwapPointer(&s.pub, unsafe.Pointer(sp)))
	if oldsp != nil {
		oldsp.cancel()
	}

	seqmerge := mergeSeqhdr{
		cb: func(pkt av.Packet) {
			sp.gc.put(pkt)
			s.notifySub()
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pkt, err := r.ReadPacket()
		if err != nil {
			return
		}

		seqmerge.do(pkt)
	}
}

type streams struct {
	l sync.RWMutex
	m map[string]*stream
}

func newStreams() *streams {
	return &streams{
		m: map[string]*stream{},
	}
}

func (ss *streams) add(k string) (*stream, func()) {
	ss.l.Lock()
	defer ss.l.Unlock()

	common.LogDebugln("[stream]", k, "add")

	s, ok := ss.m[k]
	if !ok {
		s = &stream{}
		ss.m[k] = s
	}
	s.n++

	return s, func() {
		common.LogDebugln("[stream]", k, "remove")

		ss.l.Lock()
		defer ss.l.Unlock()

		s.n--
		if s.n == 0 {
			delete(ss.m, k)
		}
	}
}

func (ss *streams) get(k string) *stream {
	ss.l.Lock()
	defer ss.l.Unlock()
	common.LogDebugln("[stream]", k, "get")
	return ss.m[k]
}

var (
	_streams *streams
)

func startRtmp(listenAddr string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	s := rtmp.NewServer()
	_streams = newStreams()
	// handle flags
	s.LogEvent = func(c *rtmp.Conn, nc net.Conn, e int) {
		es := rtmp.EventString[e]
		common.LogDebugln("RtmpEvent", nc.LocalAddr(), nc.RemoteAddr(), es)
	}

	s.HandleConn = func(c *rtmp.Conn, nc net.Conn) {
		trimmed := strings.Trim(c.URL.Path, "/")
		trimmed = strings.Replace(trimmed, "/"+settings.StreamKey, "", 1)
		stream, remove := _streams.add(trimmed)
		defer remove()

		if c.Publishing {
			pub(c, nc, stream)
		}
	}

	for {
		nc, err := lis.Accept()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		go s.HandleNetConn(nc)
	}
}

func pub(c *rtmp.Conn, nc net.Conn, s *stream) {
	streamKey := strings.Split(strings.Trim(c.URL.RequestURI(), "/"), "/")[1]
	if streamKey != "" {
		if streamKey == settings.StreamKey {
			s.setPub(c)
		} else {
			common.LogInfoln("[stream] Invalid stream key")
			nc.Close()
		}
	}
	common.LogInfoln("[stream] No stream key supplied")
	nc.Close()
}

func handleLive(w http.ResponseWriter, r *http.Request) {
	s := _streams.get(strings.Trim(r.URL.Path, "/"))
	if s != nil {
		w.Header().Set("Content-Type", "video/x-flv")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)
		flusher.Flush()
		muxer := flv.NewMuxer(writeFlusher{httpflusher: flusher, Writer: w})
		ss := &streamSub{
			notify: make(chan struct{}, 1),
		}

		s.sub.Store(ss, nil)
		defer s.sub.Delete(ss)

		var cursor *gopCacheReadCursor
		var lastsp *streamPub

		seqsplit := splitSeqhdr{
			cb: func(pkt av.Packet) error {
				return muxer.WritePacket(pkt)
			},
		}

		for {
			var pkts []av.Packet

			sp := (*streamPub)(atomic.LoadPointer(&s.pub))
			if sp != lastsp {
				cursor = &gopCacheReadCursor{}
				lastsp = sp
			}
			if sp != nil {
				cur := sp.gc.curSnapshot()
				if cur != nil {
					pkts = cursor.advance(cur)
				}
			}

			if len(pkts) != 0 {
				for _, pkt := range pkts {
					if err := seqsplit.do(pkt); err != nil {
						return
					}
				}
			}
		}
	} else {
		// Maybe HTTP_204 is better than HTTP_404
		w.WriteHeader(http.StatusNoContent)
		stats.resetViewers()
	}
}
