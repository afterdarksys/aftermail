package webrtc

import (
	"context"
	"fmt"
	"log"
)

// Options specify the WebRTC track parameters
type Options struct {
	EnableVideo bool
	EnableAudio bool
	EnableScreen bool
}

// Client manages the WebRTC peer connection
type Client struct {
	PeerDID string
}

// NewClient initializes a WebRTC connection manager targeting a local/remote DID
func NewClient(peerDID string) *Client {
	return &Client{PeerDID: peerDID}
}

// Dial establishes a P2P signaling challenge across the ALPN QUIC transport and negotiates ICE candidates
func (c *Client) Dial(ctx context.Context, opts Options) error {
	log.Printf("[WebRTC] Initiating signaling to %s over AfterSMTP QUIC multiplex...", c.PeerDID)
	// STUB: Wrap `pion/webrtc.NewPeerConnection` and exchange SDP descriptions.
	// If the Peer accepts, it triggers media track negotiation.
	return fmt.Errorf("WebRTC signaling requires active AfterSMTP gRPC bi-directional stream")
}

// HandleIncoming listens for SDP offers emitted over the QUIC stream
func (c *Client) HandleIncoming(sdpOffer []byte) ([]byte, error) {
	// STUB: Parse the offer, construct a local `pion/webrtc.PeerConnection`, add camera/screen tracks, and return the SDP Answer.
	return nil, fmt.Errorf("Incoming WebRTC handling not yet bound to local media devices")
}
