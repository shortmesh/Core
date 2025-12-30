package main

import (
	"encoding/json"
	"fmt"
	"os"
	_ "sherlock/matrix/docs"
	"sync"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	// "maunium.net/go/mautrix/id"
)

type User struct {
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	AccessToken      string `yaml:"access_token"`
	RecoveryKey      string `yaml:"recovery_key"`
	DeviceId         string `yaml:"device_id"`
	HomeServer       string `yaml:"homeserver"`
	HomeServerDomain string `yaml:"homeserver_domain"`
}

func main() {
	// ks.Init()
	conf, err := cfg.getConf()

	user := User{
		Username:         conf.User.Username,
		AccessToken:      conf.User.AccessToken,
		RecoveryKey:      conf.User.RecoveryKey,
		HomeServer:       conf.HomeServer,
		HomeServerDomain: conf.HomeServerDomain,
	}

	client, err := mautrix.NewClient(
		user.HomeServer,
		id.NewUserID(user.Username, user.HomeServerDomain),
		user.AccessToken,
	)
	client.DeviceID = id.DeviceID(conf.User.DeviceId)

	mc := MatrixClient{
		Client: client,
	}

	if len(os.Args) > 2 && os.Args[2] == "--login" {
		fmt.Println("[+] Login commencing...")
		password := conf.User.Password

		if _, err := mc.Login(password); err != nil {
			panic(err)
		}

		return
	}

	if err != nil {
		panic(err)
	}

	go SyncUser(&mc, user)

	select {}
}

func SyncUser(mc *MatrixClient, user User) {
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan *event.Event)

	go func() {
		for {
			evt := <-ch
			// fmt.Printf("%s\n", evt.Content.AsEncrypted().Ciphertext)
			json, err := json.MarshalIndent(evt, "", "")

			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", json)
			// fmt.Printf("%s\n", evt.Type)
		}
	}()

	err := mc.Sync(ch, user.RecoveryKey)
	if err != nil {
		panic(err)
	}
}
