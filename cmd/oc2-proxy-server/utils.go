package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync/atomic"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type loggingWriter struct {
	http.ResponseWriter
	reqNum int
}

func (lw *loggingWriter) WriteHeader(code int) {
	log.Printf("[%d] WriteHeader(%d)", lw.reqNum, code)
	lw.ResponseWriter.WriteHeader(code)
}

func newLoggingWriter(w http.ResponseWriter, reqNum int) *loggingWriter {
	return &loggingWriter{ResponseWriter: w, reqNum: reqNum}
}

type LoggingHandler struct {
	http.Handler
	reqCount int64
}

func (lh *LoggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqNum := int(atomic.AddInt64(&lh.reqCount, 1))
	log.Printf("[%d] %s %s %s headers=%#v", reqNum, r.RemoteAddr, r.Method, r.RequestURI, r.Header)
	lw := newLoggingWriter(w, reqNum)
	lh.Handler.ServeHTTP(lw, r)
}
