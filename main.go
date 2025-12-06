package main

import (
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat:   time.RFC3339,
		DisableHTMLEscape: true,
	})
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peer1, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	peer2, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	dc1, err := peer1.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	msgReceived := make(chan struct{})
	dc1.OnMessage(func(msg webrtc.DataChannelMessage) {
		logrus.Infof("Peer1 received message: %s", string(msg.Data))
		close(msgReceived)
	})

	peer2.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			logrus.Info("Peer2 data channel open, sending message to Peer1")
			dc.SendText("Hello from Peer2!")
		})
	})

	// lambda to wait for ICE gathering to complete
	waitICE := func(pc *webrtc.PeerConnection) {
		done := make(chan struct{})
		pc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
			if state == webrtc.ICEGathererStateComplete {
				close(done)
			}
		})
		if pc.ICEGatheringState() == webrtc.ICEGatheringStateComplete {
			return
		}
		<-done
	}

	// Peer1 creates offer
	offer, err := peer1.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	err = peer1.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}
	waitICE(peer1)

	// Peer2 sets offer as remote description
	err = peer2.SetRemoteDescription(*peer1.LocalDescription())
	if err != nil {
		panic(err)
	}

	// Peer2 creates answer
	answer, err := peer2.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	err = peer2.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}
	waitICE(peer2)

	// Peer1 sets answer as remote description
	err = peer1.SetRemoteDescription(*peer2.LocalDescription())
	if err != nil {
		panic(err)
	}

	logrus.Info("Offer/Answer exchange complete between two peers")
	select {} // Keep the program running and communication open
}
