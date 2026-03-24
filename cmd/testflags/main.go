package main

import (
	"context"
	"fmt"
	"os"

	"github.com/viamrobotics/webrtc/v3"
	"go.uber.org/zap"
	"go.viam.com/utils/rpc"
)

const (
	host    = "machine-main.ozy75nuoux.viam.cloud"
	entity  = "4f04c6ca-b9df-440d-82db-6d383b0c92a1"
	apiKey  = "ymw99jc6h6ki2m9rlhzekp8r2tpj9zjy"
	sigAddr = "app.viam.com:443"
)

var (
	pass, fail int
)

func baseOpts(extra rpc.DialWebRTCOptions) []rpc.DialOption {
	extra.SignalingAuthEntity = entity
	extra.SignalingCreds = rpc.Credentials{Type: "api-key", Payload: apiKey}
	return []rpc.DialOption{
		rpc.WithEntityCredentials(entity, rpc.Credentials{Type: "api-key", Payload: apiKey}),
		rpc.WithWebRTCOptions(extra),
	}
}

func localCandidateType(conn rpc.ClientConn) (webrtc.ICECandidateType, bool) {
	pc := conn.PeerConn()
	if pc == nil {
		return webrtc.ICECandidateType(0), false
	}
	sctp := pc.SCTP()
	if sctp == nil {
		return webrtc.ICECandidateType(0), false
	}
	dtls := sctp.Transport()
	if dtls == nil {
		return webrtc.ICECandidateType(0), false
	}
	ice := dtls.ICETransport()
	if ice == nil {
		return webrtc.ICECandidateType(0), false
	}
	pair, err := ice.GetSelectedCandidatePair()
	if err != nil || pair == nil {
		return webrtc.ICECandidateType(0), false
	}
	return pair.Local.Typ, true
}

// wantType: "relay", "!relay", or "none"
func assert(testName string, opts rpc.DialWebRTCOptions, wantType string) {
	fmt.Printf("\n=== %s ===\n", testName)

	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := rpc.DialWebRTC(ctx, sigAddr, host, sugar, baseOpts(opts)...)
	if err != nil {
		if wantType == "none" {
			fmt.Printf("PASS: connection failed (expected)\n")
			pass++
		} else {
			fmt.Printf("FAIL: unexpected connection failure: %v\n", err)
			fail++
		}
		return
	}
	defer conn.Close()

	if wantType == "none" {
		fmt.Printf("FAIL: expected connection failure, but connected\n")
		fail++
		return
	}

	typ, ok := localCandidateType(conn)
	if !ok {
		fmt.Printf("FAIL: connected but could not determine local candidate type\n")
		fail++
		return
	}

	fmt.Printf("local candidate type: %s\n", typ)

	switch wantType {
	case "relay":
		if typ == webrtc.ICECandidateTypeRelay {
			fmt.Printf("PASS: local candidate type: %s\n", typ)
			pass++
		} else {
			fmt.Printf("FAIL: expected relay, got: %s\n", typ)
			fail++
		}
	case "!relay":
		if typ != webrtc.ICECandidateTypeRelay {
			fmt.Printf("PASS: local candidate type: %s (not relay)\n", typ)
			pass++
		} else {
			fmt.Printf("FAIL: expected non-relay candidate, got: %s\n", typ)
			fail++
		}
	}
}

func main() {
	// 1. Baseline — expect non-relay (host or srflx)
	assert("baseline (no flags)", rpc.DialWebRTCOptions{}, "!relay")

	// 2. ForceRelay — expect relay
	assert("ForceRelay", rpc.DialWebRTCOptions{ForceRelay: true}, "relay")

	// 3. ForceP2P — expect non-relay (host or srflx)
	assert("ForceP2P", rpc.DialWebRTCOptions{ForceP2P: true}, "!relay")

	fmt.Printf("\n%d passed, %d failed.\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}
