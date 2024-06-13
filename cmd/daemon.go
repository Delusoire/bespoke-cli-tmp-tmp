/*
 * Copyright (C) 2024 Delusoire
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package cmd

import (
	"fmt"
	"log"
	"net/http"
	"spicetify/paths"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gorilla/websocket"
)

var (
	daemon bool
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if daemon {
			fmt.Println("Starting daemon")
			startDaemon()
		}
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting daemon")
		startDaemon()
	},
}

var daemonEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if daemon {
			fmt.Println("Daemon already enabled")
			return
		}
		fmt.Println("Enabling daemon")
		daemon = true
		viper.Set("daemon", daemon)
		viper.WriteConfig()
	},
}

var daemonDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Disabling daemon")
		daemon = false
		viper.Set("daemon", daemon)
		viper.WriteConfig()
	},
}

func init() {
	cobra.OnInitialize(func() {
		viper.SetDefault("daemon", true)
		daemon = viper.GetBool("daemon")
	})

	rootCmd.AddCommand(daemonCmd)

	daemonCmd.AddCommand(daemonStartCmd, daemonEnableCmd, daemonDisableCmd)

	viper.SetDefault("daemon", false)
}

func startDaemon() {
	viper.OnConfigChange(func(in fsnotify.Event) {
		daemon = viper.GetBool("daemon")
	})
	go viper.WatchConfig()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
	}
	defer watcher.Close()

	c := make(chan struct{})

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Create) {
					if strings.HasSuffix(event.Name, "xpui.spa") {
						if err := execApply(); err != nil {
							log.Println(err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				log.Println("error:", err)
			default:
				if !daemon {
					close(c)
				}
			}

		}
	}()

	if err := watcher.Add(paths.GetSpotifyAppsPath(spotifyDataPath)); err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/rpc", handleWebSocketProtocol)
	log.Panicln(http.ListenAndServe("localhost:7967", nil))

	<-c
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: improve security
		return true
	},
}

func handleWebSocketProtocol(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()

	for {
		_, p, err := c.ReadMessage()
		if err != nil {
			log.Println("!read:", err)
			break
		}

		incoming := string(p)
		log.Println("recv:", incoming)
		res, err := HandleProtocol(incoming)
		if err != nil {
			log.Println("!handle:", err)
		}
		c.WriteMessage(websocket.TextMessage, []byte(res))
	}
}

/*
func startDaemon() {
	viper.OnConfigChange(func(in fsnotify.Event) {
		daemon = viper.GetBool("daemon")
	})
	go viper.WatchConfig()

	ticker := time.NewTicker(5 * time.Minute)
	stop := make(chan bool)

	_, apps := getApps()
	xpuiIndex := filepath.Join(apps, "xpui", "index.html")

	go func() {
		for {
			select {
			case <-ticker.C:
				if _, err := os.Stat(xpuiIndex); err == nil {
					continue
				}
				execApply()
			default:
				if !daemon {
					stop <- true
					return
				}
			}
		}
	}()

	<-stop
}
*/
