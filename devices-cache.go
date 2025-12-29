package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"maunium.net/go/mautrix/id"
)

type Keystore struct {
	connection *sql.DB
	filepath   string
}

func (ks *Keystore) Init() {
	db, err := sql.Open("sqlite3", ks.filepath)
	if err != nil {
		panic(err)
	}
	ks.connection = db

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users ( 
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		username TEXT NOT NULL UNIQUE, 
		accessToken TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)

	if err != nil {
		panic(err)
	}
}

func (ks *Keystore) CreateUser(username string, accessToken string) error {
	tx, err := ks.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO users (username, accessToken) values(?,?)`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(username, accessToken)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (ks *Keystore) FetchUser(username string) (Users, error) {
	stmt, err := ks.connection.Prepare("select id, username, accessToken from users where username = ?")
	if err != nil {
		return Users{}, err
	}

	defer stmt.Close()

	var id int
	var _username string
	var _accessToken string
	err = stmt.QueryRow(username).Scan(&id, &_username, &_accessToken)
	if err != nil {
		return Users{}, err
	}

	return Users{ID: id, Username: _username, AccessToken: _accessToken}, nil
}

func (ks *Keystore) FetchAllUsers() ([]Users, error) {
	stmt, err := ks.connection.Prepare("select id, username, accessToken from users")
	if err != nil {
		return []Users{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return []Users{}, err
	}

	defer rows.Close()

	var users []Users
	for rows.Next() {
		var id int
		var _username string
		var _accessToken string

		err = rows.Scan(&id, &_username, &_accessToken)
		if err != nil {
			return []Users{}, err
		}

		users = append(users, Users{ID: id, Username: _username, AccessToken: _accessToken})
	}

	return users, nil
}

// https://github.com/mattn/go-sqlite3/blob/v1.14.28/_example/simple/simple.go
func (clientDb *ClientDB) Init() error {
	db, err := sql.Open("sqlite3", clientDb.filepath)
	if err != nil {
		return err
	}

	clientDb.connection = db

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS clients ( 
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	username TEXT NOT NULL UNIQUE, 
	password TEXT NOT NULL,
	accessToken TEXT NOT NULL, 
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rooms ( 
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	clientUsername TEXT NOT NULL,
	roomID TEXT NOT NULL,
	platformName TEXT NOT NULL,
	members TEXT NOT NULL,
	deviceName TEXT,
	isBridge INTEGER NOT NULL,
	sessions BLOB,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, 
	sessionsTimestamp DATETIME DEFAULT CURRENT_TIMESTAMP, 
	UNIQUE(clientUsername, roomID, platformName, isBridge)
	);
	
	CREATE TABLE IF NOT EXISTS webhooks (
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	clientUsername TEXT NOT NULL,
	deviceName TEXT NOT NULL,
	url TEXT NOT NULL,
	method TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, 
	UNIQUE(clientUsername, deviceName, url, method)
	);
	`)

	if err != nil {
		return err
	}
	return err
}

func (clientDb *ClientDB) AuthenticateAccessToken(username string, accessToken string) (bool, error) {
	query := `SELECT COUNT(*) FROM clients WHERE username = ? AND accessToken = ?`

	var count int
	err := clientDb.connection.QueryRow(query, username, accessToken).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("authentication query failed: %w", err)
	}

	if count == 0 {
		log.Printf("[-] Authentication failed for user: %s", username)
		return false, nil
	}

	log.Printf("[+] Authentication successful for user: %s", username)
	return true, nil
}

func (clientDb *ClientDB) Authenticate(username string, password string) (bool, error) {
	query := `SELECT COUNT(*) FROM clients WHERE username = ? AND password = ?`

	var count int
	err := clientDb.connection.QueryRow(query, username, password).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[-] Authentication failed for user: %s", username)
			return false, nil
		}
		return false, fmt.Errorf("authentication query failed: %w", err)
	}

	if count == 0 {
		log.Printf("[-] Authentication failed for user: %s", username)
		return false, nil
	}

	log.Printf("[+] Authentication successful for user: %s", username)
	return true, nil
}

func (clientDb *ClientDB) Store(accessToken string, password string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO clients (username, accessToken, password, timestamp) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(clientDb.username, accessToken, password)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (clientDb *ClientDB) Fetch() (string, error) {
	fmt.Println("- Fetching username:", clientDb.filepath)
	stmt, err := clientDb.connection.Prepare("select accessToken from clients where username = ?")
	var accessToken string

	if err != nil {
		return accessToken, err
	}

	defer stmt.Close()

	err = stmt.QueryRow(clientDb.username).Scan(&accessToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return accessToken, nil
		}
		return accessToken, err
	}
	return accessToken, err
}

func (clientDb *ClientDB) Close() {
	defer clientDb.connection.Close()
}

func (clientDb *ClientDB) StoreRooms(
	roomID string,
	platformName string,
	deviceName string,
	members string,
	isBridge bool,
) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO rooms (clientUsername, roomID, platformName, deviceName, members, isBridge) values(?,?,?,?,?,?)`,
	)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(clientDb.username, roomID, platformName, deviceName, members, isBridge)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to store room: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (clientDb *ClientDB) FetchRooms(roomID string) (Rooms, error) {
	stmt, err := clientDb.connection.Prepare(
		"select clientUsername, roomID, platformName, deviceName, members, isBridge from rooms where roomID = ?",
	)
	if err != nil {
		return Rooms{}, err
	}
	var clientUsername string
	var _roomID string
	var platformName string
	var deviceName string
	var members string
	var isBridge bool

	defer stmt.Close()

	err = stmt.QueryRow(roomID).Scan(&clientUsername, &_roomID, &platformName, &deviceName, &members, &isBridge)
	if err != nil {
		if err == sql.ErrNoRows {
			return Rooms{}, nil
		}
		return Rooms{}, err
	}

	var room = Rooms{
		ID:         id.RoomID(_roomID),
		isBridge:   isBridge,
		DeviceName: deviceName,
		Members: map[string]string{
			platformName: members,
		},
	}

	return room, err
}

func (clientDb *ClientDB) FetchRoomsByMembers(members string) ([]Rooms, error) {
	log.Println("Fetching room members for", members, clientDb.filepath)
	stmt, err := clientDb.connection.Prepare(
		"select clientUsername, roomID, platformName, deviceName, members, isBridge from rooms where members = ?",
	)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(members)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []Rooms
	for rows.Next() {
		var clientUsername string
		var _roomID string
		var _platformName string
		var _deviceName string
		var _members string
		var isBridge bool

		err = rows.Scan(&clientUsername, &_roomID, &_platformName, &_deviceName, &_members, &isBridge)
		if err != nil {
			return nil, err
		}

		room := Rooms{
			ID:         id.RoomID(_roomID),
			isBridge:   isBridge,
			DeviceName: _deviceName,
			Members: map[string]string{
				_platformName: _members,
			},
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

func (clientDb *ClientDB) FetchBridgeRooms(username string) ([]*Bridges, error) {
	// log.Println("Fetching bridge rooms for", username, clientDb.filepath)
	stmt, err := clientDb.connection.Prepare(
		"select clientUsername, roomID, platformName, deviceName, members, isBridge from rooms where clientUsername = ? and isBridge = 1",
	)
	if err != nil {
		return []*Bridges{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(username)
	if err != nil {
		return []*Bridges{}, err
	}

	defer rows.Close()

	var bridges = []*Bridges{}
	for rows.Next() {
		var clientUsername string
		var _roomID string
		var platformName string
		var deviceName string
		var members string
		var isBridge bool

		err = rows.Scan(&clientUsername, &_roomID, &platformName, &deviceName, &members, &isBridge)
		if err != nil {
			return []*Bridges{}, err
		}

		bridges = append(bridges, &Bridges{
			RoomID:     id.RoomID(_roomID),
			Name:       platformName,
			DeviceName: deviceName,
			BotName:    members,
		})
	}

	return bridges, err
}

func (clientDb *ClientDB) FetchActiveSessions(username string) ([]byte, time.Time, error) {
	stmt, err := clientDb.connection.Prepare("select sessions, sessionsTimestamp from rooms where clientUsername = ? and isBridge = 1")
	if err != nil {
		return []byte{}, time.Time{}, err
	}
	defer stmt.Close()

	var sessions []byte
	var sessionsTimestamp time.Time
	err = stmt.QueryRow(username).Scan(&sessions, &sessionsTimestamp)
	if err != nil {
		return []byte{}, time.Time{}, err
	}
	return sessions, sessionsTimestamp, nil
}

func (clientDb *ClientDB) StoreActiveSessions(username string, sessions []byte) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE rooms SET sessions = ?, sessionsTimestamp = CURRENT_TIMESTAMP WHERE clientUsername = ? AND isBridge = 1")
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(sessions, username)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (clientDb *ClientDB) RemoveActiveSessions(username string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("update rooms set sessions = NULL, sessionsTimestamp = CURRENT_TIMESTAMP where clientUsername = ?")
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(username)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func IsActiveSessionsExpired(clientDb *ClientDB, username string) bool {
	sessions, sessionsTimestamp, err := clientDb.FetchActiveSessions(username)
	if err != nil {
		return true
	}

	if sessionsTimestamp.IsZero() || len(sessions) == 0 {
		return true
	}

	now := time.Now()
	return now.After(sessionsTimestamp.Add(16 * time.Second))
}

// Webhook CRUD methods

// CreateWebhook creates a new webhook entry
func (clientDb *ClientDB) CreateWebhook(deviceName string, url string, method string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO webhooks (clientUsername, deviceName, url, method, timestamp) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(clientDb.username, deviceName, url, method)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FetchWebhook retrieves a webhook by device name, URL, and method
func (clientDb *ClientDB) FetchWebhook(deviceName string, url string, method string) (Webhook, error) {
	stmt, err := clientDb.connection.Prepare(`
		SELECT id, clientUsername, deviceName, url, method, timestamp 
		FROM webhooks 
		WHERE clientUsername = ? AND deviceName = ? AND url = ? AND method = ?
	`)
	if err != nil {
		return Webhook{}, err
	}

	defer stmt.Close()

	var id int
	var clientUsername string
	var _deviceName string
	var _url string
	var _method string
	var timestamp time.Time

	err = stmt.QueryRow(clientDb.username, deviceName, url, method).Scan(&id, &clientUsername, &_deviceName, &_url, &_method, &timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return Webhook{}, nil
		}
		return Webhook{}, err
	}

	return Webhook{
		ID:             id,
		ClientUsername: clientUsername,
		DeviceName:     _deviceName,
		URL:            _url,
		Method:         _method,
	}, nil
}

// FetchWebhooksByDevice retrieves all webhooks for a specific device
func (clientDb *ClientDB) FetchWebhooksByDevice(deviceName string) ([]Webhook, error) {
	stmt, err := clientDb.connection.Prepare(`
		SELECT id, clientUsername, deviceName, url, method, timestamp 
		FROM webhooks 
		WHERE clientUsername = ? AND deviceName = ?
	`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(clientDb.username, deviceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var id int
		var clientUsername string
		var _deviceName string
		var url string
		var method string
		var timestamp time.Time

		err = rows.Scan(&id, &clientUsername, &_deviceName, &url, &method, &timestamp)
		if err != nil {
			return nil, err
		}

		webhook := Webhook{
			ID:             id,
			ClientUsername: clientUsername,
			DeviceName:     _deviceName,
			URL:            url,
			Method:         method,
		}
		webhooks = append(webhooks, webhook)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return webhooks, nil
}

// FetchAllWebhooks retrieves all webhooks for the current user
func (clientDb *ClientDB) FetchAllWebhooks() ([]Webhook, error) {
	stmt, err := clientDb.connection.Prepare(`
		SELECT id, clientUsername, deviceName, url, method, timestamp 
		FROM webhooks 
		WHERE clientUsername = ?
	`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(clientDb.username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var id int
		var clientUsername string
		var deviceName string
		var url string
		var method string
		var timestamp time.Time

		err = rows.Scan(&id, &clientUsername, &deviceName, &url, &method, &timestamp)
		if err != nil {
			return nil, err
		}

		webhook := Webhook{
			ID:             id,
			ClientUsername: clientUsername,
			DeviceName:     deviceName,
			URL:            url,
			Method:         method,
		}
		webhooks = append(webhooks, webhook)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return webhooks, nil
}

// UpdateWebhook updates an existing webhook
func (clientDb *ClientDB) UpdateWebhook(deviceName string, oldURL string, oldMethod string, newURL string, newMethod string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		UPDATE webhooks 
		SET url = ?, method = ?, timestamp = CURRENT_TIMESTAMP 
		WHERE clientUsername = ? AND deviceName = ? AND url = ? AND method = ?
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	result, err := stmt.Exec(newURL, newMethod, clientDb.username, deviceName, oldURL, oldMethod)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("no webhook found to update")
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteWebhook deletes a specific webhook
func (clientDb *ClientDB) DeleteWebhook(deviceName string, url string, method string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		DELETE FROM webhooks 
		WHERE clientUsername = ? AND deviceName = ? AND url = ? AND method = ?
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	result, err := stmt.Exec(clientDb.username, deviceName, url, method)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("no webhook found to delete")
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteWebhooksByDevice deletes all webhooks for a specific device
func (clientDb *ClientDB) DeleteWebhooksByDevice(deviceName string) error {
	tx, err := clientDb.connection.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		DELETE FROM webhooks 
		WHERE clientUsername = ? AND deviceName = ?
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(clientDb.username, deviceName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete webhooks for device: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
