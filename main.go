package main

import (
	_ "sherlock/matrix/docs"
	// "maunium.net/go/mautrix/id"
)

func main() {
	ks.Init()

	go func() {
		err := (&MatrixClient{}).SyncAllClients()
		if err != nil {
			panic(err)
		}
	}()
}
