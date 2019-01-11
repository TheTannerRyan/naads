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
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

// HTTP starts an endpoint for viewing the status of the NAADS client.
func (c *Client) HTTP(port int) {
	go func() {
		// parse status page
		page, err := template.ParseFiles("status.html")
		if err != nil {
			log.Printf("CONTROLLER [ERROR] Unable to read status.html")
		}

		// register HTTP route
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// generate data for page
			data := c.generateStatus()
			if err := page.ExecuteTemplate(w, "status", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		// start endpoint
		log.Fatalln(http.ListenAndServe(":"+strconv.Itoa(port), nil))
	}()
}

// status is for rendering the HTTP status page
type status struct {
	Version    string
	Uptime     string
	Time       string
	UTCTime    string
	FeedStatus []feedstatus
	FeedConfig []feedconfig
}

// feedstatus is for rendering the HTTP status page
type feedstatus struct {
	Status         string
	StatusStyle    string
	Name           string
	Host           string
	LastMsg        string
	LastMsgTime    string
	Disconnections string
}

// feedconfig is for rendering the HTTP status page
type feedconfig struct {
	Name            string
	Host            string
	SendHeartbeat   string
	ConnectTimeout  string
	LivenessTimeout string
	ReconnectDelay  string
	Logging         string
	LogHeartbeat    string
}

// generateStatus generates a status struct for rendering the status template.
func (c *Client) generateStatus() *status {
	// system data
	currentTime := time.Now()
	currentUptime := currentTime.Sub(c.startTime)
	uptimeStr := fmt.Sprintf("%d days %d hours %d minutes %d seconds",
		int(currentUptime.Hours()/24),
		int(currentUptime.Hours())%24,
		int(currentUptime.Minutes())%60,
		int(currentUptime.Seconds())%60)

	var feedStatus []feedstatus
	var feedConfig []feedconfig

	for i, f := range c.Feeds {
		// feed status
		status := feedstatus{}
		if f.isConnected {
			if c.activeFeed == i {
				status.Status = "LOCKED"
				status.StatusStyle = "status-locked"
			} else {
				status.Status = "ACTIVE"
				status.StatusStyle = "status-active"
			}
		} else {
			status.Status = "DOWN"
			status.StatusStyle = "status-down"
		}
		status.Name = f.Name
		status.Host = f.Host
		status.LastMsg = f.lastMsg
		if diff := int(time.Now().Sub(f.lastMsgTime).Seconds()); diff == 9223372036 {
			status.LastMsgTime = "N/A"
		} else {
			status.LastMsgTime = "(" + strconv.Itoa(diff) + " seconds ago)"
		}
		status.Disconnections = strconv.Itoa(f.disconnections)
		feedStatus = append(feedStatus, status)

		// feed config
		config := feedconfig{}
		config.Name = f.Name
		config.Host = f.Host
		if f.SendHeartbeat {
			config.SendHeartbeat = "YES"
		} else {
			config.SendHeartbeat = "NO"
		}
		config.ConnectTimeout = strconv.Itoa(int(f.ConnectTimeout.Seconds())) + "s"
		config.LivenessTimeout = strconv.Itoa(int(f.LivenessTimeout.Seconds())) + "s"
		config.ReconnectDelay = strconv.Itoa(int(f.ReconnectDelay.Seconds())) + "s"
		if f.Logging {
			config.Logging = "YES"
		} else {
			config.Logging = "NO"
		}
		if f.LogHeartbeat {
			config.LogHeartbeat = "YES"
		} else {
			config.LogHeartbeat = "NO"
		}
		feedConfig = append(feedConfig, config)
	}

	// construct status struct
	return &status{
		Version:    version,
		Uptime:     uptimeStr,
		Time:       currentTime.Format("Mon Jan 2 2006 15:04:05 MST"),
		UTCTime:    currentTime.UTC().Format("Mon Jan 2 2006 15:04:05 MST"),
		FeedStatus: feedStatus,
		FeedConfig: feedConfig,
	}
}
