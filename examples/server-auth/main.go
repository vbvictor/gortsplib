// Package main contains an example.
package main

import (
	"log"
	"sync"

	"github.com/pion/rtp"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/liberrors"
)

// This example shows how to
// 1. create a RTSP server which accepts plain connections.
// 2. allow a single client to publish a stream, if it provides credentials.
// 3. allow several clients to read the stream, if they provide credentials.

const (
	// credentials required to publish the stream
	publishUser = "publishuser"
	publishPass = "publishpass"

	// credentials required to read the stream
	readUser = "readuser"
	readPass = "readpass"
)

type serverHandler struct {
	server    *gortsplib.Server
	mutex     sync.RWMutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
}

// called when a connection is opened.
func (sh *serverHandler) OnConnOpen(_ *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Printf("conn opened")
}

// called when a connection is closed.
func (sh *serverHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Printf("conn closed (%v)", ctx.Error)
}

// called when a session is opened.
func (sh *serverHandler) OnSessionOpen(_ *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Printf("session opened")
}

// called when a session is closed.
func (sh *serverHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Printf("session closed")

	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	// if the session is the publisher,
	// close the stream and disconnect any reader.
	if sh.stream != nil && ctx.Session == sh.publisher {
		sh.stream.Close()
		sh.stream = nil
	}
}

// called when receiving a DESCRIBE request.
func (sh *serverHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (
	*base.Response, *gortsplib.ServerStream, error,
) {
	log.Printf("DESCRIBE request")

	// Verify reader credentials.
	// In case of readers, credentials have to be verified during DESCRIBE and SETUP.
	ok := ctx.Conn.VerifyCredentials(ctx.Request, readUser, readPass)
	if !ok {
		return &base.Response{
			StatusCode: base.StatusUnauthorized,
		}, nil, liberrors.ErrServerAuth{}
	}

	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	// no one is publishing yet
	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send medias that are being published to the client
	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

// called when receiving an ANNOUNCE request.
func (sh *serverHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Printf("ANNOUNCE request")

	// Verify publisher credentials.
	// In case of publishers, credentials have to be verified during ANNOUNCE.
	ok := ctx.Conn.VerifyCredentials(ctx.Request, publishUser, publishPass)
	if !ok {
		return &base.Response{
			StatusCode: base.StatusUnauthorized,
		}, liberrors.ErrServerAuth{}
	}

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	// disconnect existing publisher
	if sh.stream != nil {
		sh.stream.Close()
		sh.publisher.Close()
	}

	// create the stream and save the publisher
	sh.stream = &gortsplib.ServerStream{
		Server: sh.server,
		Desc:   ctx.Description,
	}
	err := sh.stream.Initialize()
	if err != nil {
		panic(err)
	}
	sh.publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

// called when receiving a SETUP request.
func (sh *serverHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (
	*base.Response, *gortsplib.ServerStream, error,
) {
	log.Printf("SETUP request")

	// SETUP is used by both readers and publishers. In case of publishers, just return StatusOK.
	if ctx.Session.State() == gortsplib.ServerSessionStatePreRecord {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil, nil
	}

	// Verify reader credentials.
	// In case of readers, credentials have to be verified during DESCRIBE and SETUP.
	ok := ctx.Conn.VerifyCredentials(ctx.Request, readUser, readPass)
	if !ok {
		return &base.Response{
			StatusCode: base.StatusUnauthorized,
		}, nil, liberrors.ErrServerAuth{}
	}

	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	// no one is publishing yet
	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

// called when receiving a PLAY request.
func (sh *serverHandler) OnPlay(_ *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Printf("PLAY request")

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

// called when receiving a RECORD request.
func (sh *serverHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Printf("RECORD request")

	// called when receiving a RTP packet
	ctx.Session.OnPacketRTPAny(func(medi *description.Media, _ format.Format, pkt *rtp.Packet) {
		// route the RTP packet to all readers
		err := sh.stream.WritePacketRTP(medi, pkt)
		if err != nil {
			log.Printf("ERR: %v", err)
		}
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func main() {
	// configure the server
	h := &serverHandler{}
	h.server = &gortsplib.Server{
		Handler:           h,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	// start server and wait until a fatal error
	log.Printf("server is ready on %s", h.server.RTSPAddress)
	panic(h.server.StartAndWait())
}
