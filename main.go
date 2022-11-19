package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bananenpro/cli"
	"github.com/adrg/xdg"
	"github.com/code-game-project/go-client/cg"
	"github.com/code-game-project/go-utils/sessions"
)

var gameSessionDir = filepath.Join(xdg.DataHome, "codegame", "games")

func logMessage(severity cg.DebugSeverity, message string, data string) {
	defer cli.PrintColor(cli.Cyan, "%s%s%s", strings.Repeat("-", 39), time.Now().Format("2006-01-02 15:04:05"), strings.Repeat("-", 40))

	var color cli.Color
	switch severity {
	case cg.DebugTrace:
		color = "\x1b[2m"
		cli.PrintColor(color, "TRACE: %s", message)
	case cg.DebugInfo:
		color = ""
		cli.PrintColor(color, "INFO: %s", message)
	case cg.DebugWarning:
		color = "\x1b[33m"
		cli.PrintColor(color, "WARNING: %s", message)
	case cg.DebugError:
		color = "\x1b[1;31m"
		cli.PrintColor(color, "ERROR: %s", message)
	}
	if data == "" {
		return
	}

	formatted, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err == nil {
		cli.PrintColor(color, string(formatted))
	} else {
		cli.PrintColor(color, data)
	}
}

func debugServer(socket *cg.DebugSocket) error {
	cli.Clear()
	cli.PrintColor(cli.CyanBold, "%s%s%s", strings.Repeat("=", 40), " Server Debug Log ", strings.Repeat("=", 40))
	err := socket.DebugServer()
	if err != nil {
		return errors.New("failed to connect to server")
	}
	return nil
}

func debugGame(socket *cg.DebugSocket) error {
	gameId, err := cli.Input("Game ID:")
	if err != nil {
		return err
	}
	cli.Clear()
	cli.PrintColor(cli.CyanBold, "%s%s%s", strings.Repeat("=", 40), "= Game Debug Log =", strings.Repeat("=", 40))
	err = socket.DebugGame(gameId)
	if err != nil {
		return errors.New("failed to connect to game")
	}
	return nil
}

func debugPlayer(socket *cg.DebugSocket) error {
	var fromSessionStorage bool
	var err error
	if _, err := os.Stat(filepath.Join(gameSessionDir, url.PathEscape(socket.URL()))); err == nil {
		fromSessionStorage, err = cli.YesNo("Select player from session storage", false)
		if err != nil {
			return err
		}
	}

	var gameId string
	var playerId string
	var playerSecret string
	if fromSessionStorage {
		gameId, playerId, playerSecret, err = selectFromSessionStorage(socket)
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
	err = socket.DebugPlayer(gameId, playerId, playerSecret)
	if err != nil {
		return errors.New("failed to connect to player")
	}
	return nil
}

func selectFromSessionStorage(socket *cg.DebugSocket) (gameId string, playerId string, playerSecret string, err error) {
	users, err := sessions.ListUsernames(socket.URL())
	if err != nil {
		return "", "", "", err
	}

	if len(users) == 0 {
		return "", "", "", errors.New("no sessions available")
	}

	index, err := cli.Select("User:", users)
	if err != nil {
		return "", "", "", err
	}

	sessionData, err := os.ReadFile(filepath.Join(gameSessionDir, url.PathEscape(socket.URL()), users[index]+".json"))
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
