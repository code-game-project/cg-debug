package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bananenpro/cli"
	"github.com/adrg/xdg"
	"github.com/code-game-project/go-client/cg"
)

func logMessage(severity cg.DebugSeverity, message string, data string) {
	defer cli.PrintColor(cli.Cyan, "%s%s%s", strings.Repeat("-", 39), time.Now().Format("2006-01-02 15:04:05"), strings.Repeat("-", 40))

	switch severity {
	case cg.DebugTrace:
		cli.Print("\x1b[2mTRACE: %s", message)
	case cg.DebugInfo:
		cli.Print("INFO: %s", message)
	case cg.DebugWarning:
		cli.Print("\x1b[33mWARNING: %s", message)
	case cg.DebugError:
		cli.Print("\x1b[1;31mERROR: %s", message)
	}
	if data == "" {
		return
	}

	formatted, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err == nil {
		cli.PrintColor("", string(formatted))
	} else {
		cli.PrintColor("", data)
	}
}

func debugServer(socket *cg.DebugSocket) error {
	cli.Clear()
	cli.PrintColor(cli.CyanBold, "%s%s%s", strings.Repeat("=", 40), " Server Debug Log ", strings.Repeat("=", 40))
	return socket.DebugServer()
}

func debugGame(socket *cg.DebugSocket) error {
	gameId, err := cli.Input("Game ID:")
	if err != nil {
		return err
	}
	cli.Clear()
	cli.PrintColor(cli.CyanBold, "%s%s%s", strings.Repeat("=", 40), "= Game Debug Log =", strings.Repeat("=", 40))
	return socket.DebugGame(gameId)
}

func debugPlayer(socket *cg.DebugSocket) error {
	yes, err := cli.YesNo("Select player from session storage", false)
	if err != nil {
		return err
	}

	var gameId string
	var playerId string
	var playerSecret string
	if yes {
		gameId, playerId, playerSecret, err = selectFromSessionStorage()
		if err != nil {
			return err
		}
	} else {
		gameId, err = cli.Input("Game ID:")
		if err != nil {
			return err
		}
		playerId, err = cli.Input("Player ID:")
		if err != nil {
			return err
		}
		playerSecret, err = cli.Input("Player secret:")
		if err != nil {
			return err
		}
	}

	cli.Clear()
	cli.PrintColor(cli.CyanBold, "%s%s%s", strings.Repeat("=", 40), " Player Debug Log ", strings.Repeat("=", 40))
	return socket.DebugPlayer(gameId, playerId, playerSecret)
}

func selectFromSessionStorage() (gameId string, playerId string, playerSecret string, err error) {
	gamesPath := filepath.Join(xdg.DataHome, "codegame", "games")

	gameDirs, err := os.ReadDir(gamesPath)
	if err != nil {
		return "", "", "", err
	}

	games := make([]string, 0, len(gameDirs))
	for _, dir := range gameDirs {
		if dir.IsDir() {
			games = append(games, dir.Name())
		}
	}

	index, err := cli.Select("Game:", games)
	if err != nil {
		return "", "", "", err
	}
	game := games[index]

	userFiles, err := os.ReadDir(filepath.Join(gamesPath, game))
	if err != nil {
		return "", "", "", err
	}
	users := make([]string, 0, len(userFiles))
	for _, dir := range userFiles {
		if !dir.IsDir() && strings.HasSuffix(dir.Name(), ".json") {
			users = append(users, string(dir.Name()[:len(dir.Name())-5]))
		}
	}

	index, err = cli.Select("User:", users)
	if err != nil {
		return "", "", "", err
	}

	sessionData, err := os.ReadFile(filepath.Join(gamesPath, game, users[index]+".json"))
	if err != nil {
		return "", "", "", err
	}

	var session cg.Session
	err = json.Unmarshal(sessionData, &session)
	if err != nil {
		return "", "", "", err
	}

	return session.GameId, session.PlayerId, session.PlayerSecret, nil
}

func main() {
	var url string
	var err error
	if len(os.Args) < 2 {
		url, err = cli.Input("Game server URL:")
		if err != nil {
			return
		}
	} else {
		url = os.Args[1]
	}

	severities, err := cli.MultiSelect("Severities:", []string{"Trace", "Info", "Warning", "Error"}, []int{1, 2, 3})
	if err != nil {
		return
	}

	socket := cg.NewDebugSocket(url)
	socket.SetSeverities(severities[0], severities[1], severities[2], severities[3])
	socket.OnMessage(logMessage)

	target, err := cli.Select("Target:", []string{"Server", "Game", "Player"})
	if err != nil {
		return
	}

	switch target {
	case 0:
		err = debugServer(socket)
	case 1:
		err = debugGame(socket)
	case 2:
		err = debugPlayer(socket)
	}
	if err != nil {
		if err != cli.ErrCanceled {
			cli.Error(err.Error())
		}
		return
	}
}
