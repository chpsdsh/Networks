package gui

import (
	"fmt"
	"snake-game/internal/application/client"
	"snake-game/internal/application/game"
	"snake-game/internal/domain"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type GameView struct {
	app    fyne.App
	window fyne.Window

	client *client.Client

	boardWidget *BoardWidget

	games     []domain.DiscoveredGame
	gamesList *widget.List

	currentGame   *game.Game
	players       []domain.PlayerInfo
	ratingList    *widget.List
	gameInfoLabel *widget.Label

	availableGamesList *widget.List
}

func NewFyneView(a fyne.App, w fyne.Window, client *client.Client) *GameView {
	return &GameView{
		app:    a,
		window: w,
		client: client,
	}
}

// показ стартового окна
func (v *GameView) ShowStartMenu() {
	fyne.Do(func() {
		v.window.Canvas().SetOnTypedKey(nil)
		startContent := container.NewVSplit(
			widget.NewButton("New game", func() {
				v.client.CreateGame()
			}),
			widget.NewButton("Join game", func() {
				v.client.ShowGameList()
			}),
		)
		v.window.SetContent(startContent)
		v.window.Resize(fyne.NewSize(500, 300))
	})
}

// ShowError — отображает ошибку пользователю в GUI
func (v *GameView) ShowError(msg string) {
	if msg == "" || v.window == nil {
		return
	}

	fyne.Do(func() {
		dialog.ShowError(
			fmt.Errorf("%s", msg),
			v.window,
		)
	})
}

func (v *GameView) ShowGames() {
	v.window.Canvas().SetOnTypedKey(nil)

	v.games = v.client.GamesSnapshot()

	v.gamesList = widget.NewList(
		func() int { return len(v.games) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(v.games) {
				return
			}
			g := v.games[i]
			text := fmt.Sprintf(
				"%s [%s]  %d игроков  %dx%d  %s",
				g.GameName,
				g.Host,
				g.Players,
				g.Width,
				g.Height,
				map[bool]string{true: "Can join", false: "View only"}[g.CanJoin],
			)
			o.(*widget.Label).SetText(text)
		},
	)

	v.gamesList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(v.games) {
			return
		}
		selected := v.games[id]
		v.showJoinDialog(selected)
		v.gamesList.Unselect(id)
	}

	backBtn := widget.NewButton("Back", func() {
		v.client.BackToStart()
	})

	refreshBtn := widget.NewButton("Refresh", func() {
		v.client.ShowGameList()
	})

	gamesScroll := container.NewVScroll(v.gamesList)
	gamesScroll.SetMinSize(fyne.NewSize(260, 200))

	content := container.NewBorder(
		widget.NewLabel("Games"),
		container.NewVBox(refreshBtn, backBtn),
		nil, nil,
		gamesScroll,
	)

	v.window.SetContent(content)
}

func (v *GameView) RefreshRating() {
	if v.ratingList == nil || v.currentGame == nil {
		return
	}

	fyne.Do(func() {
		v.players = v.currentGame.PlayersSnapshot()
		v.ratingList.Refresh()
		v.refreshGameInfo()
	})
}

func (v *GameView) RefreshGamesList() {
	if v.gamesList == nil {
		return
	}

	fyne.Do(func() {
		v.games = v.client.GamesSnapshot()
		v.gamesList.Refresh()
	})
}

func (v *GameView) RefreshAvailableGames() {
	if v.availableGamesList == nil {
		return
	}

	fyne.Do(func() {
		v.games = v.client.GamesSnapshot()
		v.availableGamesList.Refresh()
	})
}

func (v *GameView) showJoinDialog(g domain.DiscoveredGame) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("name")

	var selectedMode = "normal"

	normalRadio := widget.NewRadioGroup([]string{"Player (normal)", "Viewer"}, func(selected string) {
		switch selected {
		case "Player (normal)":
			selectedMode = "normal"
		case "Viewer":
			selectedMode = "viewer"
		}
	})
	normalRadio.SetSelected("Player (normal)")

	form := widget.NewForm(
		widget.NewFormItem("To Game", widget.NewLabel(g.GameName)),
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Mode", normalRadio),
	)

	dialog.ShowCustomConfirm(
		"Join game",
		"OK",
		"Return",
		form,
		func(ok bool) {
			if !ok {
				return
			}
			name := nameEntry.Text
			if name == "" {
				dialog.ShowError(fmt.Errorf("input name"), v.window)
				return
			} else {
				if err := v.client.JoinGame(name, g, selectedMode); err != nil {
					dialog.ShowError(err, v.window)
					return
				}
			}

		},
		v.window,
	)
}

func (v *GameView) ShowConfigMenu() {
	v.window.Canvas().SetOnTypedKey(nil)
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Nickname")

	widthEntry := widget.NewEntry()
	widthEntry.SetPlaceHolder("Width (10-100)")

	heightEntry := widget.NewEntry()
	heightEntry.SetPlaceHolder("Height (10-100)")

	foodEntry := widget.NewEntry()
	foodEntry.SetPlaceHolder("Food Quantity (0-100)")

	delayEntry := widget.NewEntry()
	delayEntry.SetPlaceHolder("Delay, ms (100-3000)")

	form := widget.NewForm(
		widget.NewFormItem("Nickname", nameEntry),
		widget.NewFormItem("Width", widthEntry),
		widget.NewFormItem("Height", heightEntry),
		widget.NewFormItem("Food Quantity", foodEntry),
		widget.NewFormItem("Delay, ms", delayEntry),
	)

	form.OnSubmit = func() {
		name := nameEntry.Text

		w, err1 := strconv.Atoi(widthEntry.Text)
		h, err2 := strconv.Atoi(heightEntry.Text)
		food, err3 := strconv.Atoi(foodEntry.Text)
		delay, err4 := strconv.Atoi(delayEntry.Text)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			dialog.ShowError(fmt.Errorf("not valid game parameters"), v.window)
			return
		}

		cfg := domain.GameConfig{
			Width:        w,
			Height:       h,
			FoodStatic:   food,
			StateDelayMs: delay,
		}

		if err := v.client.CreateNewGame(name, cfg); err != nil {
			v.ShowError(err.Error())
			return
		}
	}

	form.OnCancel = func() {
		v.client.BackToStart()
	}

	v.window.SetContent(container.NewVBox(
		widget.NewLabel("Create New Game"),
		form,
	))

}

func (v *GameView) ShowGameScreen(game *game.Game) {
	fyne.Do(func() {
		v.currentGame = game

		// слева – поле
		v.boardWidget = NewBoardWidget(game)
		boardContainer := container.NewStack(v.boardWidget)

		// ==== справа – панель ====

		// Рейтинг
		ratingTitle := widget.NewLabel("Рейтинг")

		v.players = v.currentGame.PlayersSnapshot()

		v.ratingList = widget.NewList(
			func() int { return len(v.players) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				if i < 0 || i >= len(v.players) {
					return
				}
				p := v.players[i]
				o.(*widget.Label).SetText(
					fmt.Sprintf("%d. %s (%d)", i+1, p.Name, p.Score),
				)
			},
		)
		ratingBox := container.NewVBox(ratingTitle, v.ratingList)

		// Текущая игра — реальные значения
		currentGameTitle := widget.NewLabel("Current game")

		cfg := game.Config() // сделай метод Config() в домене, который возвращает GameConfig

		v.gameInfoLabel = widget.NewLabel(
			fmt.Sprintf("Owner: ?\nSize: %dx%d\nFood: %d+1x", cfg.Width, cfg.Height, cfg.FoodStatic),
		)
		v.refreshGameInfo()
		currentGameBox := container.NewVBox(currentGameTitle, v.gameInfoLabel)

		// Кнопки "Выход" и "Новая игра"
		exitBtn := widget.NewButton("Exit", func() {
			v.client.LeaveGame()
		})
		newGameBtn := widget.NewButton("New Game", func() {
			v.client.NewGameFromInGame()
		})
		buttonsRow := container.NewHBox(exitBtn, newGameBtn)

		// Список доступных игр (пока можно оставить заглушкой или убрать)
		gamesTitle := widget.NewLabel("Available Games")

		v.games = v.client.GamesSnapshot()

		v.availableGamesList = widget.NewList(
			func() int { return len(v.games) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				if i < 0 || i >= len(v.games) {
					return
				}
				g := v.games[i]
				o.(*widget.Label).SetText(fmt.Sprintf(
					"%s [%s]  %d players  %dx%d  %s",
					g.GameName,
					g.Host,
					g.Players,
					g.Width,
					g.Height,
					map[bool]string{true: "Can join", false: "View only"}[g.CanJoin],
				))
			},
		)

		v.availableGamesList.OnSelected = func(id widget.ListItemID) {
			if id < 0 || id >= len(v.games) {
				return
			}
			selected := v.games[id]
			v.showJoinDialog(selected)
			v.availableGamesList.Unselect(id)
		}

		gamesScroll := container.NewVScroll(v.availableGamesList)
		gamesScroll.SetMinSize(fyne.NewSize(260, 200)) // 200px по высоте — уже видно несколько элементов

		gamesBox := container.NewVBox(gamesTitle, gamesScroll)

		// правая колонка
		topRight := container.NewVBox(
			ratingBox,
			currentGameBox,
			buttonsRow,
		)

		// gamesBox у тебя уже VBox(title + scroll)
		right := container.NewPadded(
			container.NewBorder(
				topRight, // top
				nil,      // bottom
				nil,      // left
				nil,      // right
				gamesBox, // center: займёт всё оставшееся место
			),
		)
		content := container.NewBorder(nil, nil, nil, right, boardContainer)
		v.window.SetContent(content)

		v.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			switch ev.Name {
			case fyne.KeyW, fyne.KeyUp:
				v.client.ChangeDirection(domain.DirUp)
			case fyne.KeyS, fyne.KeyDown:
				v.client.ChangeDirection(domain.DirDown)
			case fyne.KeyA, fyne.KeyLeft:
				v.client.ChangeDirection(domain.DirLeft)
			case fyne.KeyD, fyne.KeyRight:
				v.client.ChangeDirection(domain.DirRight)
			}
		})
	})
}
func (v *GameView) RefreshBoard() {
	if v.boardWidget == nil {
		return
	}

	fyne.Do(func() {
		v.boardWidget.Refresh()
		v.refreshGameInfo()
		v.RefreshAvailableGames()
	})

}

func (v *GameView) refreshGameInfo() {
	if v.currentGame == nil || v.gameInfoLabel == nil {
		return
	}
	cfg := v.currentGame.Config()
	master := v.currentGame.MasterName()
	if master == "" {
		master = "?"
	}
	v.gameInfoLabel.SetText(
		fmt.Sprintf("Owner: %s\nSize: %dx%d\nFood: %d+1x",
			master, cfg.Width, cfg.Height, cfg.FoodStatic,
		),
	)
}
