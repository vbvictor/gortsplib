package rtph264

import (
	"bytes"
	"testing"

	"github.com/pion/rtp"
	"github.com/stretchr/testify/require"
)

func uint16Ptr(v uint16) *uint16 {
	return &v
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func mergeBytes(vals ...[]byte) []byte {
	size := 0
	for _, v := range vals {
		size += len(v)
	}
	res := make([]byte, size)

	pos := 0
	for _, v := range vals {
		n := copy(res[pos:], v)
		pos += n
	}

	return res
}

var cases = []struct {
	name  string
	nalus [][]byte
	pkts  []*rtp.Packet
}{
	{
		"single",
		[][]byte{
			mergeBytes(
				[]byte{0x05},
				bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 8),
			),
		},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x05},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 8),
				),
			},
		},
	},
	{
		"fragmented",
		[][]byte{
			mergeBytes(
				[]byte{0x05},
				bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 187),
			),
		},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         false,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x85},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 124),
					[]byte{0, 1, 2, 3, 4, 5},
				),
			},
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17646,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x45},
					[]byte{6, 7},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 62),
				),
			},
		},
	},
	{
		"fragmented to the limit",
		[][]byte{bytes.Repeat([]byte{1}, 1997)},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         false,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x81},
					bytes.Repeat([]byte{1}, 998),
				),
			},
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17646,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x41},
					bytes.Repeat([]byte{1}, 998),
				),
			},
		},
	},
	{
		"aggregated",
		[][]byte{
			{0x09, 0xF0},
			{
				0x41, 0x9a, 0x24, 0x6c, 0x41, 0x4f, 0xfe, 0xd6,
				0x8c, 0xb0, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x6d, 0x40,
			},
		},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: []byte{
					0x18, 0x00, 0x02, 0x09,
					0xf0, 0x00, 0x44, 0x41, 0x9a, 0x24, 0x6c, 0x41,
					0x4f, 0xfe, 0xd6, 0x8c, 0xb0, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
					0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
					0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
					0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
					0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x6d, 0x40,
				},
			},
		},
	},
	{
		"aggregated followed by single",
		[][]byte{
			{0x09, 0xF0},
			{
				0x41, 0x9a, 0x24, 0x6c, 0x41, 0x4f, 0xfe, 0xd6,
				0x8c, 0xb0, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
				0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x6d, 0x40,
			},
			mergeBytes(
				[]byte{0x08},
				bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 120),
			),
		},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         false,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: []byte{
					0x18, 0x00, 0x02, 0x09,
					0xf0, 0x00, 0x44, 0x41, 0x9a, 0x24, 0x6c, 0x41,
					0x4f, 0xfe, 0xd6, 0x8c, 0xb0, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
					0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
					0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00,
					0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00,
					0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03,
					0x00, 0x00, 0x03, 0x00, 0x00, 0x6d, 0x40,
				},
			},
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17646,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x08},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 120),
				),
			},
		},
	},
	{
		"fragmented followed by aggregated",
		[][]byte{
			mergeBytes(
				[]byte{0x05},
				bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 187),
			),
			{0x09, 0xF0},
			{0x09, 0xF0},
		},
		[]*rtp.Packet{
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         false,
					PayloadType:    96,
					SequenceNumber: 17645,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x85},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 124),
					[]byte{0, 1, 2, 3, 4, 5},
				),
			},
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         false,
					PayloadType:    96,
					SequenceNumber: 17646,
					SSRC:           0x9dbb7812,
				},
				Payload: mergeBytes(
					[]byte{0x1c, 0x45},
					[]byte{6, 7},
					bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 62),
				),
			},
			{
				Header: rtp.Header{
					Version:        2,
					Marker:         true,
					PayloadType:    96,
					SequenceNumber: 17647,
					SSRC:           0x9dbb7812,
				},
				Payload: []byte{
					0x18, 0x00, 0x02, 0x09,
					0xf0, 0x00, 0x02, 0x09, 0xf0,
				},
			},
		},
	},
}

func TestEncode(t *testing.T) {
	for _, ca := range cases {
		t.Run(ca.name, func(t *testing.T) {
			e := &Encoder{
				PayloadType:           96,
				SSRC:                  uint32Ptr(0x9dbb7812),
				InitialSequenceNumber: uint16Ptr(0x44ed),
				PayloadMaxSize:        1000,
			}
			err := e.Init()
			require.NoError(t, err)

			pkts, err := e.Encode(ca.nalus)
			require.NoError(t, err)
			require.Equal(t, ca.pkts, pkts)
		})
	}
}

func TestEncodeRandomInitialState(t *testing.T) {
	e := &Encoder{
		PayloadType: 96,
	}
	err := e.Init()
	require.NoError(t, err)
	require.NotEqual(t, nil, e.SSRC)
	require.NotEqual(t, nil, e.InitialSequenceNumber)
}
