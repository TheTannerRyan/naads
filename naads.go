// Copyright (c) 2019 Tanner Ryan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package naads

import (
	"log"
	"os"
	"time"

	"github.com/thetannerryan/cap"
)

const version = "v0.0.3" // NAADS client version

// Client represents the configuration for the NAAD client.
type Client struct {
	Feeds      []*Feed         // Array of NAADS Feeds to listen to (feeds defined first have greater priority when multiple feeds are available)
	Logging    bool            // Indicator to log control status to stdout
	ch         chan *cap.Alert // Alert output channel
	activeFeed int             // Index of active feed
	startTime  time.Time       // Start time of client
}

// Start will start the highly available NAADS client. It will connect to all
// the feeds listed, locking to one of the feeds. When locked, the feed's
// messages will be passed to the output channel. If the locked feed goes down,
// the client will automatically lock onto another available feed. The
// individual feeds are responsible for providing their connection status, and
// for performing reconnect procedures.
func (c *Client) Start() chan *cap.Alert {
	// log.Printf to stdout
	log.SetOutput(os.Stdout)
	// initially no feeds are locked
	c.activeFeed = -1
	// master output feed
	c.ch = make(chan *cap.Alert, 16)
	// update start time
	c.startTime = time.Now()

	// start each feed in a goroutine (feed has it's own subclient)
	for index, feed := range c.Feeds {
		go func(i int, f *Feed) {
			// range over single feed's output channel
			for alert := range f.start() {
				// forward message to output channel only if the feed is locked
				// as the active feed
				if i == c.activeFeed {
					c.ch <- alert
				}
			}
		}(index, feed)
	}
	// begin monitoring the feeds
	c.monitor()
	// return the Alert output channel
	return c.ch
}

// monitor is responsible for continuously monitoring the health of the feeds.
// If the current locked feed is down, or if there are no available feeds, it
// will continue searching for feeds until a feed is available (and locked).
func (c *Client) monitor() {
	go func() {
		for {
			// initial delay + check health every second
			time.Sleep(1 * time.Second)

			if c.activeFeed == -1 {
				// currently not locked to feed
				feedIndex := c.findAvailableFeed()
				if feedIndex == -1 {
					log.Printf("CONTROL [ERROR] ALL FEEDS ARE DEAD !!\n")
				} else {
					// lock the new feed
					c.activeFeed = feedIndex
					currentFeed := c.Feeds[c.activeFeed]
					log.Printf("CONTROL [STATUS] Successfully locked feed to %s\n", currentFeed.Name)
				}
			} else {
				// currently locked to feed, check health and switch if
				// necessary
				currentFeed := c.Feeds[c.activeFeed]
				if !currentFeed.isConnected {
					// current feed is down, find another feed
					c.activeFeed = -1
					feedIndex := c.findAvailableFeed()
					if feedIndex == -1 {
						// unlock the active feed
						log.Printf("CONTROL [ERROR] ALL FEEDS ARE DEAD !!\n")
					} else {
						// lock the new feed
						c.activeFeed = feedIndex
						currentFeed := c.Feeds[c.activeFeed]
						log.Printf("CONTROL [STATUS] Successfully locked feed to %s\n", currentFeed.Name)
					}
				}
			}
		}
	}()
}

// findAvailableFeed returns the index of the first feed in Feeds that is
// connected. If there are no feeds that are connected, -1 is returned.
func (c *Client) findAvailableFeed() int {
	for index, feed := range c.Feeds {
		if feed.isConnected {
			return index
		}
	}
	return -1
}
