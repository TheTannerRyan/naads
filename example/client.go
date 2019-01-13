// Copyright (c) 2019 Tanner Ryan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"github.com/thetannerryan/naads"
)

func main() {
	// Create a new naads client. Upon initial connection, if both feeds are
	// successfully connected, the client will lock onto the first feed.
	client := &naads.Client{
		Feeds: []*naads.Feed{
			{
				Name:            "NAADS-1",
				Host:            "streaming1.naad-adna.pelmorex.com",
				SendHeartbeat:   true,
				ConnectTimeout:  1 * time.Second,
				LivenessTimeout: 65 * time.Second,
				ReconnectDelay:  21 * time.Second,
				Logging:         true,
				LogHeartbeat:    true,
			},
			{
				Name:            "NAADS-2",
				Host:            "streaming2.naad-adna.pelmorex.com",
				SendHeartbeat:   true,
				ConnectTimeout:  1 * time.Second,
				LivenessTimeout: 65 * time.Second,
				ReconnectDelay:  21 * time.Second,
				Logging:         true,
				LogHeartbeat:    true,
			},
		},
		Logging: true,
	}

	// start HTTP server on port 6060
	client.HTTP(6060)

	// receive the alerts (highly available)
	for alert := range client.Start() {
		fmt.Println("EXAMPLE CLIENT (sender): " + alert.Sender)
	}
}
