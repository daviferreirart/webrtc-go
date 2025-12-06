package main

import (
	"fmt"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
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

	dc1.OnMessage(func(msg webrtc.DataChannelMessage) {
		logrus.Infof("Peer1 received message: %s", string(msg.Data))
	})

	peer2.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			logrus.Info("Peer2 data channel open, sending message to Peer1")
			for i := 0; i < 1000; i++ {
				time.Sleep(10 * time.Second)
				dc.SendText(fmt.Sprintf("Hello from Peer2! %d", i))
			}
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

	// Periodically log stats from peer1
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := peer1.GetStats()
			for id, stat := range stats {
				if codecStat, ok := stat.(webrtc.CodecStats); ok {
					logrus.Infof("Codec Stat [%s]: %+v", id, codecStat)
				}
				if inboundRTPStat, ok := stat.(webrtc.InboundRTPStreamStats); ok {
					logrus.Infof("Inbound RTP Stat [%s]: %+v", id, inboundRTPStat)
				}
				if outboundRTPStat, ok := stat.(webrtc.OutboundRTPStreamStats); ok {
					logrus.Infof("Outbound RTP Stat [%s]: %+v", id, outboundRTPStat)
				}
				if dataChannelStat, ok := stat.(webrtc.DataChannelStats); ok {
					logrus.Infof("Data Channel Stat [%s]: %+v", id, dataChannelStat)
				}

				if transportStat, ok := stat.(webrtc.TransportStats); ok {
					logrus.Infof("Transport Stat [%s]: %+v", id, transportStat)
				}

			}
		}
	}()

	select {} // Keep the program running and communication open
}
