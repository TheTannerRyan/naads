// BSD 2-Clause License
//
// Copyright (c) 2019 Tanner Ryan. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package naads

import (
	"bytes"
	"log"
	"net"
	"os"
	"time"

	"github.com/thetannerryan/cap"
)

var (
	startSignature = []byte("<alert")   // byte sequence signifying start of alert
	endSignature   = []byte("</alert>") // byte sequence signifying end of alert
)

// Feed is a TCP client for the NAADS system. It will be used for receiving the
// TCP data stream and for converting the raw XML to CAP Alert structs.
type Feed struct {
	Name            string          // Name of NAADS server (display purposes)
	Host            string          // Hostname of NAADS server
	SendHeartbeat   bool            // Send NAADS heartbeats to output channel
	ConnectTimeout  time.Duration   // Timeout on connection/reconnection
	LivenessTimeout time.Duration   // Duration between messages before feed is considered dead
	ReconnectDelay  time.Duration   // Delay before attempting reconnection
	Logging         bool            // Indicator to log feed status to stdout
	LogHeartbeat    bool            // If logging is enabled, indicator to log heartbeats to stdout
	ch              chan *cap.Alert // Alert output channel
	isConnected     bool            // Indicator if connection is currently established
	lastMsgTime     time.Time       // Last time a message (alert or heartbeat) was received (not currently used)
	lastMsg         string          // Type and ID of last message that was received
	disconnections  int             // Number of times the feed was disconnected
}

// start will establish a connection with the NAADS server (via internal
// connect) and return an Alert output channel. The feed will automatically
// perform health checks and perform reconnects as necessary.
func (feed *Feed) start() chan *cap.Alert {
	// log.Printf to stdout
	log.SetOutput(os.Stdout)
	// create an output channel for alerts
	feed.ch = make(chan *cap.Alert, 16)
	feed.connect()
	return feed.ch
}

// connect is the internal function that spawns a goroutine, responsible for
// connecting to the TCP stream. It listens to the NAAD Host, converting the raw
// XML data into valid Alert structs. The function will also call itself to
// initialize reconnects.
func (feed *Feed) connect() {
	// connect() is non-blocking
	go func(f *Feed) {
		// Establish connection with host. Wait ConnectTimeout before the
		// connection attempt is considered failed.
		dial := &net.Dialer{Timeout: f.ConnectTimeout}
		conn, err := dial.Dial("tcp", f.Host+":8080")
		if err != nil {
			// Error was encountered when performing connection attempt. Update
			// status and wait ReconnectDelay before re-attempting connection.
			f.isConnected = false
			if f.Logging {
				log.Printf("%s [ERROR] Cannot establish connection with %s; waiting %.f seconds and retrying\n", f.Name, f.Host, f.ReconnectDelay.Seconds())
			}
			time.Sleep(f.ReconnectDelay)
			f.connect()
			return
		}

		// Temp buffer for chunks (protocol uses max of 5MB); data buffer for
		// storing entire message.
		temp := make([]byte, 6*1024*1024)
		data := make([]byte, 0)

		// if block is reached, feed was successfully connected
		f.isConnected = true
		if f.Logging {
			log.Printf("%s [STATUS] Established connection with %s\n", f.Name, f.Host)
		}

		for {
			// Connection is considered dead if we don't receive messages after
			// the feed defined LivenessTimeout.
			conn.SetDeadline(time.Now().Add(f.LivenessTimeout))

			// stream data to temp buffer
			n, err := conn.Read(temp)
			if err != nil {
				// connection was dropped
				f.isConnected = false
				f.disconnections++
				// Ensure the connection actually closes (prevent resource leak
				// with conn.SetDeadline).
				if err2 := conn.Close(); err2 != nil {
				}
				if f.Logging {
					log.Printf("%s [ERROR] Lost connection with %s; attempting reconnection\n", f.Name, f.Host)
				}
				time.Sleep(f.ConnectTimeout)
				f.connect()
				return
			}

			// normal operation: if start signature encountered, clear data
			// buffer for new data
			startIndex := bytes.Index(temp, startSignature)
			if startIndex != -1 {
				if f.Logging {
					if startIndex == 38 {
						if f.LogHeartbeat {
							f.lastMsg = "HEARTBEAT "
							log.Printf("%s [STATUS] INCOMING HEARTBEAT\n", f.Name)
						}
					} else if startIndex == 55 {
						f.lastMsg = "ALERT "
						log.Printf("%s [STATUS] INCOMING ALERT\n", f.Name)
					} else {
						f.lastMsg = "UNKNOWN "
						log.Printf("%s [STATUS] INCOMING UNKNOWN\n", f.Name)
					}
				}
				// clear data buffer
				data = data[:0]
			}

			// normal operation: append last chunk of temp buffer to data buffer
			lastChunk := temp[:n]
			data = append(data, lastChunk...)

			// normal operation: if end signature encountered, the data is ready
			// to be parsed
			endIndex := bytes.Index(lastChunk, endSignature)
			if endIndex != -1 {
				// parse the message using CAP package
				alert, err := cap.ParseCAP(data)
				if err != nil {
					//> TODO: incorporate manually fetching of malformed +
					// missing messages
					if f.Logging {
						log.Printf("%s [ERROR] MALFORMED MESSAGE\n", f.Name)
					}
				} else {
					// update message time; broadcast message on channel
					f.lastMsg += alert.Identifier
					f.lastMsgTime = time.Now()
					if f.SendHeartbeat || (!f.SendHeartbeat && alert.Sender != "NAADS-Heartbeat") {
						f.ch <- alert
					}
				}
			}
		}
	}(feed)
}
