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
	"log"
	"os"
	"time"

	"github.com/thetannerryan/cap"
)

// Client represents the configuration for the NAAD client.
type Client struct {
	Feeds      []*Feed         // Array of NAADS Feeds to listen to
	Logging    bool            // Indicator to log control status to stdout
	ch         chan *cap.Alert // Alert output channel
	activeFeed int             // Index of active feed
}

// Start will establish connections with all of the provided feeds. It will
// designate one of the feeds as "hot" and begin forwarding messages from the
// feed's output to the master output. In the event that a feed goes down, the
// built in monitor will switch to another feed.
func (c *Client) Start() chan *cap.Alert {
	// log.Printf to stdout
	log.SetOutput(os.Stdout)
	// master output feed
	c.ch = make(chan *cap.Alert, 16)

	// spawn a goroutine with each feed; start each feed
	for index, feed := range c.Feeds {
		go func(i int, f *Feed) {
			// range over single feed's output channel
			for alert := range f.start() {
				// publish alert on main channel only if current feed is the hot
				// feed
				if i == c.activeFeed {
					c.ch <- alert
				}
			}
		}(index, feed)
	}

	time.Sleep(2 * time.Second)
	if c.Logging {
		log.Printf("CONTROL [STATUS] Successfully locked feed to %s\n", c.Feeds[c.activeFeed].Name)
	}

	// monitor the feeds
	go func() {
		for {
			// get current and adjacent feed
			currentFeed := c.Feeds[c.activeFeed]
			adjIndex := (c.activeFeed + 1) % len(c.Feeds)
			adjFeed := c.Feeds[adjIndex]

			// switch to adjacent feed if current is down
			if !currentFeed.isConnected {
				if c.Logging {
					log.Printf("CONTROL [ERROR] %s is down; switching main feed to %s\n", currentFeed.Name, adjFeed.Name)
				}
				// change feed and index
				c.activeFeed = adjIndex
				currentFeed = adjFeed
				// confirm that switch was successful
				if currentFeed.isConnected && c.Logging {
					log.Printf("CONTROL [STATUS] Successfully locked feed to %s\n", currentFeed.Name)
				}
			}
			// check again in 2 seconds
			time.Sleep(2 * time.Second)
		}
	}()

	// return the master output channel
	return c.ch
}
