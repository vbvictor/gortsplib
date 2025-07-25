//go:build cgo

// Package main contains an example.
package main

import (
	"image"
	"image/jpeg"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph265"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
	"github.com/pion/rtp"
)

// This example shows how to
// 1. connect to a RTSP server.
// 2. check if there's a H265 stream.
// 3. decode the H265 stream into RGBA frames.
// 4. convert RGBA frames to JPEG images and save them on disk.

// This example requires the FFmpeg libraries, that can be installed with this command:
// apt install -y libavcodec-dev libswscale-dev gcc pkg-config

func saveToFile(img image.Image) error {
	// create file
	fname := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + ".jpg"
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.Println("saving", fname)

	// convert to jpeg
	return jpeg.Encode(f, img, &jpeg.Options{
		Quality: 60,
	})
}

func main() {
	// parse URL
	u, err := base.ParseURL("rtsp://myuser:mypass@localhost:8554/mystream")
	if err != nil {
		panic(err)
	}

	c := gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	// connect to the server
	err = c.Start2()
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// find available medias
	desc, _, err := c.Describe(u)
	if err != nil {
		panic(err)
	}

	// find the H265 media and format
	var forma *format.H265
	medi := desc.FindFormat(&forma)
	if medi == nil {
		panic("media not found")
	}

	// setup RTP -> H265 decoder
	rtpDec, err := forma.CreateDecoder()
	if err != nil {
		panic(err)
	}

	// setup H265 -> RGBA decoder
	h265Dec := &h265Decoder{}
	err = h265Dec.initialize()
	if err != nil {
		panic(err)
	}
	defer h265Dec.close()

	// if VPS, SPS and PPS are present into the SDP, send them to the decoder
	if forma.VPS != nil {
		h265Dec.decode([][]byte{forma.VPS})
	}
	if forma.SPS != nil {
		h265Dec.decode([][]byte{forma.SPS})
	}
	if forma.PPS != nil {
		h265Dec.decode([][]byte{forma.PPS})
	}

	// setup a single media
	_, err = c.Setup(desc.BaseURL, medi, 0, 0)
	if err != nil {
		panic(err)
	}

	firstRandomAccess := false
	saveCount := 0

	// called when a RTP packet arrives
	c.OnPacketRTP(medi, forma, func(pkt *rtp.Packet) {
		// extract access units from RTP packets
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			if err != rtph265.ErrNonStartingPacketAndNoPrevious && err != rtph265.ErrMorePacketsNeeded {
				log.Printf("ERR: %v", err)
			}
			return
		}

		// wait for a random access unit
		if !firstRandomAccess && !h265.IsRandomAccess(au) {
			log.Printf("waiting for a random access unit")
			return
		}
		firstRandomAccess = true

		// convert H265 access units into RGBA frames
		img, err := h265Dec.decode(au)
		if err != nil {
			panic(err)
		}

		// check for frame presence
		if img == nil {
			log.Printf("ERR: frame cannot be decoded")
			return
		}

		// convert frame to JPEG and save to file
		err = saveToFile(img)
		if err != nil {
			panic(err)
		}

		saveCount++
		if saveCount == 5 {
			log.Printf("saved 5 images, exiting")
			os.Exit(1)
		}
	})

	// start playing
	_, err = c.Play(nil)
	if err != nil {
		panic(err)
	}

	// wait until a fatal error
	panic(c.Wait())
}
