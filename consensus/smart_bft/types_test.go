package smart_bft

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProto(t *testing.T) {
	requests := make([]RequestEnvelope, 3)
	for i := range requests {
		requests[i].RequestId = strconv.Itoa(i)
		requests[i].ClientId = strconv.Itoa(i + 233)
		requests[i].SubmitterId = strconv.Itoa(i + 42)
		requests[i].SubmitterSignature = []byte("HelloWorld")
	}

	data := make([][]byte, len(requests))
	for i := range requests {
		var err error
		data[i], err = proto.Marshal(&requests[i])
		if err != nil {
			t.Fatal("cannot marshal data:", err)
		}
	}

	blockBytes := encodeBlockDataRaw(data)
	var blockRequest BlockRequest
	err := proto.Unmarshal(blockBytes, &blockRequest)
	if err != nil {
		t.Fatal("cannot encode to BlockRequest type:", err)
	}

	for i := range requests {
		rr := blockRequest.Requests[i]
		assert.Equal(t, true, rr.RequestId == strconv.Itoa(i), "request id inconsistent")
		assert.Equal(t, true, rr.ClientId == strconv.Itoa(i+233), "client id inconsistent")
		assert.Equal(t, true, rr.SubmitterId == strconv.Itoa(i+42), "submitter id inconsistent")
		assert.Equal(t, true, bytes.Equal(rr.SubmitterSignature, []byte("HelloWorld")), "signature inconsistent")
	}
}
