package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	// Postgres import
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	host = "localhost"
	port = 5432
	user = "postgres"
	// password = ""
	dbname = "postgres"
)

var db *sql.DB

func init() {
	fmt.Println("Database init")
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, os.Getenv("pqDev"), dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	sqlStatement := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL,
		userid text,
		username text,
		xp int,
		level int,
		coins int,
		PRIMARY KEY (userid)  
	  )`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	sqlStatement = `CREATE TABLE IF NOT EXISTS games (
		id SERIAL,
		name text,
		count int,
		PRIMARY KEY (name)  
	  )`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	sqlStatement = `CREATE TABLE IF NOT EXISTS servers (
		id SERIAL,
		serverid text,
		plugins int [],
		logchannel text,
		prefix text,
		PRIMARY KEY (serverid)  
	  )`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	sqlStatement = `CREATE TABLE IF NOT EXISTS reports (
		id SERIAL,
		serverid text,
		type int,
		victim text,
		mod text,
		msg text,
		PRIMARY KEY (id)  
	  )`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	go Level()
}

// CreateUser ...
func CreateUser(userid, username string) {
	sqlStatement := `
	INSERT INTO users 
	(userid, username, xp, level, coins) 
	VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(sqlStatement, userid, username, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created user %v \n", userid)
}

// UpdateUserXP ...
func UpdateUserXP(userid string) {
	sqlStatement := `
	UPDATE users
	SET xp = xp + 1
	WHERE userid = $1;`
	_, err := db.Exec(sqlStatement, userid)
	if err != nil {
		panic(err)
	}
}

// CheckUser ...
func CheckUser(userid, username string) {
	var id string
	result := db.QueryRow("SELECT userid from users where userid = $1", userid).Scan(&id)
	if result == sql.ErrNoRows {
		CreateUser(userid, username)
	} else {
		UpdateUserXP(userid)
	}
}

//Level updates the level from every user automatically
func Level() {
	start := time.Now()
	var (
		userid string
		xp     int
	)
	rows, err := db.Query("SELECT userid, xp from users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&userid, &xp)
		if err != nil {
			log.Fatal(err)
		}
		//example level
		if xp > 0 && xp < 10 {
			_, err = db.Exec("UPDATE users SET level = 1 WHERE userid = $1", userid)
			if err != nil {
				log.Fatal(err)
			}
		} else if xp > 10 && xp < 20 {
			_, err = db.Exec("UPDATE users SET level = 2 WHERE userid = $1", userid)
			if err != nil {
				log.Fatal(err)
			}
		} else if xp > 20 && xp < 30 {
			_, err = db.Exec("UPDATE users SET level = 3 WHERE userid = $1", userid)
			if err != nil {
				log.Fatal(err)
			}
		} else if xp > 30 && xp < 40 {
			_, err = db.Exec("UPDATE users SET level = 4 WHERE userid = $1", userid)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	elapsed := time.Since(start)
	log.Printf("Level update took %s", elapsed)
	// update every 60 seconds
	time.AfterFunc(time.Second*60, Level)
}

// ReturnXP returns the current xp of the specific userid
func ReturnXP(userid string) string {
	var xp string
	result := db.QueryRow("SELECT xp from users where userid = $1", userid).Scan(&xp)
	if result == sql.ErrNoRows {
		fmt.Println("Found no user")
	} else {
		return xp
	}
	return ""
}

func GetLevel(userid string) string {
	var level string
	result := db.QueryRow("SELECT level from users where userid = $1", userid).Scan(&level)
	if result == sql.ErrNoRows {
		fmt.Println("Found no user")
	} else {
		return level
	}
	return ""
}

// AddGame ...
func AddGame(name ...string) {
	for _, k := range name {
		sqlStatement := `
		INSERT INTO games 
		(name, count)
		VALUES ($1, $2)`
		_, err := db.Exec(sqlStatement, k, 0)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Launching game: %v \n", k)
	}
}

func AddCoins(userid string, coins int) {
	_, err := db.Exec("UPDATE users SET coins = coins + $2 WHERE userid = $1", userid, coins)
	if err != nil {
		log.Fatal(err)
	}
}

func GetCoins(userid string) string {
	var coins string
	result := db.QueryRow("SELECT coins from users where userid = $1", userid).Scan(&coins)
	if result == sql.ErrNoRows {
		fmt.Println("Found no user")
	} else {
		return coins
	}
	return ""
}

type UserLB struct {
	Level  int
	Coins  int
	Userid string
}

// missing server parameter
func Leaderboard() []UserLB {
	var (
		level  int
		coins  int
		userid string
	)
	slice := []UserLB{}
	//where server...
	rows, err := db.Query("SELECT level, coins, userid FROM users ORDER BY level desc, coins desc")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&level, &coins, &userid)
		if err != nil {
			log.Fatal(err)
		}
		slice = append(slice, UserLB{Level: level, Coins: coins, Userid: userid})
	}
	return slice
}

func AddPluginToServer(serverID string, pluginID int) {

	if !isPluginValid(serverID, pluginID) {
		fmt.Println(pluginID)
		_, err := db.Exec("UPDATE servers SET plugins = array_append(plugins, $1) WHERE serverid = $2", pluginID, serverID)

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Added plugin")
	}
}

func isPluginValid(serverID string, pluginID int) bool {
	var plugins pq.Int64Array
	result := db.QueryRow("SELECT plugins FROM servers WHERE serverid = $1", serverID).Scan(&plugins)
	if result == sql.ErrNoRows {
		fmt.Println("Found no plugins")
	} else {
		m := int64(pluginID)
		for _, i := range plugins {
			if m == i {
				fmt.Println("found plugin")
				return true
			}
		}
	}
	fmt.Println("return false")
	return false
}

//GetPluginsForGuild returns all plugins for a specific guild
func GetPluginsForGuild(serverID string) []int64 {
	var plugins pq.Int64Array
	result := db.QueryRow("SELECT plugins FROM servers WHERE serverid = $1", serverID).Scan(&plugins)
	if result == sql.ErrNoRows {
		fmt.Println("Found no plugins")
	} else {
		return plugins
	}
	return nil
}

func RemovePlugin(serverID string, pluginID int) {
	if isPluginValid(serverID, pluginID) {
		_, err := db.Exec("UPDATE servers SET plugins = array_remove(plugins, $1) WHERE serverid = $2", pluginID, serverID)

		if err != nil {
			log.Fatal(err)
		}
	}
}

func GetPluginForGuild(serverID string, pluginID int) int {
	var plugin int
	result := db.QueryRow("SELECT id FROM servers WHERE serverid = $1 AND $2 = ANY(plugins)", serverID, pluginID).Scan(&plugin)
	if result == sql.ErrNoRows {
		fmt.Println("Found no plugin")
		return -1
	} else {
		return plugin
	}
}

func AddLogChannel(guildID, channel string) {
	_, err := db.Exec("UPDATE servers SET logchannel = $1 WHERE serverid = $2", channel, guildID)
	if err != nil {
		log.Fatal(err)
	}
}

func ReplaceLogChannel(guildID, channel string) {
	_, err := db.Exec("UPDATE servers SET logchannel = $1 WHERE serverid = $2", channel, guildID)
	if err != nil {
		log.Fatal(err)
	}
}

func RemoveLogChannel(guildID string) {
	_, err := db.Exec("UPDATE servers SET logchannel =  NULL WHERE serverid = $1", guildID)
	if err != nil {
		log.Fatal(err)
	}
}

func GetLogChannel(guildID string) string {
	var logChannel string
	result := db.QueryRow("SELECT logchannel FROM servers WHERE serverid = $1", guildID).Scan(&logChannel)
	if result == sql.ErrNoRows {
		fmt.Println("Found no log")
		return ""
	} else {
		return logChannel
	}
}

func InitGuild(guildID string) {
	sqlStatement := `
	INSERT INTO servers 
	(serverid, plugins, logchannel, prefix)
	VALUES ($1, $2, $3, $4)`
	ar := []int{1}
	_, err := db.Exec(sqlStatement, guildID, pq.Array(ar), "", "!")
	if err != nil {
		fmt.Println(err)
	}
}

func IsGuildInDataBase(guildID string) bool {
	result := db.QueryRow("SELECT id FROM servers WHERE serverid = $1", guildID).Scan()
	if result == sql.ErrNoRows {
		fmt.Println("Found no log")
		return false
	} else {
		return true
	}
}

func GetGuildPrefix(guildID string) string {
	var prefix string
	result := db.QueryRow("SELECT prefix FROM servers WHERE serverid = $1", guildID).Scan(&prefix)
	if result == sql.ErrNoRows {
		fmt.Println("Found no log")
		return ""
	} else {
		return prefix
	}
}

func ChangePrefix(guildID, newPrefix string) {
	_, err := db.Exec("UPDATE servers SET prefix =  $1 WHERE serverid = $2", newPrefix, guildID)
	if err != nil {
		log.Fatal(err)
	}
}

func AddReport(guildID, victim, mod, msg string, ReportType int) int {
	sqlStatement := `
	INSERT INTO reports 
	(serverid, type, victim, mod, msg)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id`
	var id int
	err := db.QueryRow(sqlStatement, guildID, ReportType, victim, mod, msg).Scan(&id)
	if err != nil {
		fmt.Println(err)
	}
	return id
}

func DeleteReport(id int) {
	sqlStatement := `
	DELETE FROM reports
	WHERE id = $1;`
	_, err := db.Exec(sqlStatement, id)
	if err != nil {
		fmt.Println(err)
	}
}

type Report struct {
	ID         int
	ReportType int
	Victim     string
	Mod        string
	Msg        string
}

func GetReports(guild, victim string) []Report {
	var rep Report
	slice := []Report{}
	rows, err := db.Query("SELECT id, type, victim, mod, msg FROM reports WHERE serverid = $1 AND victim = $2", guild, victim)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&rep.ID, &rep.ReportType, &rep.Victim, &rep.Mod, &rep.Msg)
		if err != nil {
			log.Fatal(err)
		}
		slice = append(slice, rep)
	}
	return slice
}
