// Copyright (c) 2019 Tanner Ryan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
			log.Printf("CONTROLLER [ERROR]  Unable to read status.html")
		}

		mux := http.NewServeMux()
		server := &http.Server{
			Addr:         ":" + strconv.Itoa(port),
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		// register HTTP route
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// generate data for page
			data := c.generateStatus()
			if err := page.ExecuteTemplate(w, "status", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		// start endpoint
		log.Fatalln(server.ListenAndServe())
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
	Status          string
	StatusStyle     string
	Name            string
	LastMsg         string
	LastMsgTime     string
	CountDisconnect string
	CountAlert      string
	CountHeartbeat  string
	CountTest       string
	CountUnknown    string
}

// feedconfig is for rendering the HTTP status page
type feedconfig struct {
	Name            string
	Host            string
	SendHeartbeat   string
	ConnectTimeout  string
	LivenessTimeout string
	ReconnectDelay  string
	LogStatus       string
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
		status.LastMsg = f.lastMsg
		if diff := int(time.Now().Sub(f.lastMsgTime).Seconds()); diff == 9223372036 {
			status.LastMsgTime = "N/A"
		} else {
			status.LastMsgTime = "(" + strconv.Itoa(diff) + " seconds ago)"
		}
		status.CountDisconnect = strconv.Itoa(f.countDisconnect)
		status.CountAlert = strconv.Itoa(f.countAlert)
		status.CountHeartbeat = strconv.Itoa(f.countHeartbeat)
		status.CountTest = strconv.Itoa(f.countTest)
		status.CountUnknown = strconv.Itoa(f.countUnknown)
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
		if f.LogStatus {
			config.LogStatus = "YES"
		} else {
			config.LogStatus = "NO"
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
