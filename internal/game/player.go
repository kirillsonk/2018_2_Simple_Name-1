package game

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

type PlayerState struct {
	Nickname string          `json:"nickname"`
	HP       int             `json:"hp"`
	Mobs     map[string]*Mob `json:"mobs"`
	Points   int             `json:"points"`
	AllDead  bool            `json:"-"`
}

type Player struct {
	Room       *Room
	Conn       *websocket.Conn
	SingleFlag bool
	BotFlag    bool
	State      PlayerState
	Message    chan *IncommingMessage
	Listenflag chan bool
	T          *time.Timer
}

func NewPlayer(Nickname string, Conn *websocket.Conn, SingleFlag bool) *Player {
	return &Player{
		State:      PlayerState{Nickname: Nickname, Mobs: make(map[string]*Mob)},
		Conn:       Conn,
		SingleFlag: SingleFlag,
		Listenflag: make(chan bool),
		Message:    make(chan *IncommingMessage),
		BotFlag:    false,
	}
}
func GetBot() *Player {
	return &Player{
		State:      PlayerState{Nickname: "Gennadiy", Mobs: make(map[string]*Mob)},
		SingleFlag: true,
		Listenflag: make(chan bool),
		Message:    make(chan *IncommingMessage),
		BotFlag:    true,
	}
}
func (p *Player) Listen() {

	fmt.Println("Player " + p.State.Nickname + ": Start listening.")
Loop:
	for {
		select {
		case msg := <-p.Message: //читает когда приходит сообщение, асинхронно
			fmt.Println(*msg)
			fmt.Println("Player " + p.State.Nickname + ": Translate message to room.")

			p.Room.InCommand <- &IncommingCommand{InMsg: msg, Nickname: p.State.Nickname}
		case flag := <-p.Listenflag:
			switch flag {
			case false:
				if !p.BotFlag {
					p.Room.Unregister <- p
					p.Conn.Close()
				} else {
					p.T.Stop()
				}
				break Loop
			case true:

				fmt.Println("Player " + p.State.Nickname + ": Wait incomming message.")

				if !p.BotFlag {
					go func() {
						msg := &IncommingMessage{}
						err := p.Conn.ReadJSON(msg)
						if websocket.IsUnexpectedCloseError(err) {
							p.Listenflag <- false
							return
						}

						fmt.Println("Player " + p.State.Nickname + ": Get incomming message.")

						p.Message <- msg
						p.Listenflag <- true
					}()
				} else {
					p.T = time.AfterFunc(2*time.Second, func() {
						key := rand.Int()
						fmt.Println("Random key", key%3)
						msg := &IncommingMessage{}
						switch key % 3 {
						case 0:
							fmt.Println("Gennadiy killing! ")
							pos_key := rand.Int()
							var pos Position
							switch pos_key % 3 {
							case 0:
								fmt.Println("Your mob ")
								rival := p.Room.GetRival(p)
								mobs := rival.State.Mobs
								if len(mobs) != 0 {
									for _, mob := range mobs {
										pos = mob.Pos
										break
									}
									msg = new(IncommingMessage)
									msg.Command = CommandKillMob
									msg.ClickPos = pos
								} else {
									fmt.Println("No rival mob ")
									p.Listenflag <- true
									return
								}
							case 2:
								fmt.Println("Empty area")
								pos = Position{X: float64(rand.Intn(int(p.Room.AreaParams.Width))), Y: float64(rand.Intn(int(p.Room.AreaParams.Height)))}
								msg = new(IncommingMessage)
								msg.Command = CommandKillMob
								msg.ClickPos = pos
							default:
								fmt.Println("Gennadiy do not want to kill ")
								p.Listenflag <- true
								return
							}
						case 1:
							fmt.Println("Gennadiy create mob")
							mob_key := rand.Int()
							switch mob_key % 3 {
							case 0:
								msg = new(IncommingMessage)
								msg.Command = CommandAddMob
								msg.CreateMobType = "mob1"
							case 1:
								msg = new(IncommingMessage)
								msg.Command = CommandAddMob
								msg.CreateMobType = "mob2"
							case 2:
								msg = new(IncommingMessage)
								msg.Command = CommandAddMob
								msg.CreateMobType = "mob3"
							}
						default:
							fmt.Println("Gennadiy Sleep ")
							p.Listenflag <- true
							return
						}

						p.Message <- msg
						p.Listenflag <- true
						return
					})
				}
			}
		}
	}

	fmt.Println("Player " + p.State.Nickname + ": End listening.")
}

func (p *Player) Send(msg *Message) {

	// fmt.Println("Send to player "+p.State.Nickname+" ", msg.Status, " message: ")

	err := p.Conn.WriteJSON(msg)
	if err != nil {
		fmt.Println("Error send to player " + p.State.Nickname)
		p.Listenflag <- false
		return
	}
}

func (p *Player) AddMobCommand(Mobtype string) {

	fmt.Println("Player " + p.State.Nickname + ": Perform command addmob " + Mobtype)

	if CheckMobType(Mobtype) {
		mob := CreateMob(Mobtype, GetInitPos(p.Room.OwnTargetParams, p.Room.AreaParams))
		if mob.Price <= p.State.Points {
			p.State.Points -= mob.Price
			p.State.Mobs[mob.ID] = mob
		} else {
			go func() {
				p.Room.Message <- &PrivateMessage{Player: p, Msg: &Message{Status: StatusInfo, Room: p.Room.ID, Info: "Not enough points to buy."}}
			}()
		}
	}
}

func (p *Player) KillMobCommand(pos Position) int {

	fmt.Println("Player " + p.State.Nickname + ": Perform command killmob.")

	killPoints := 0
	for _, mob := range p.State.Mobs {
		if mob.Status != "dead" {
			if mob.CheckKillPos(pos) {
				mob.HP--
				if mob.HP == 0 {
					killPoints += mob.KillPoints
					mob.SetDead()
				}
			}
		}
	}

	fmt.Println("Player "+p.State.Nickname+": killPoints = ", killPoints)

	return killPoints
}

func (p *Player) ProgressState() int {
	// fmt.Println("Player " + p.State.Nickname + ": ProgressState.")
	isDead := true
	hpAttack := 0
	for _, mob := range p.State.Mobs {
		switch mob.Status {
		case "run":
			mob.ProgressState(p.Room.RivalTargetParams, p.Room.AreaParams)
			if mob.CheckTargetPos(p.Room.RivalTargetParams) {
				mob.SetAttack()
			}
		case "attack":
			hpAttack += mob.Force
		}
		if mob.Status != "dead" {
			isDead = false
		}
	}
	p.State.AllDead = isDead
	return hpAttack
}

func (p *Player) IncreasePoints(count int) {
	p.State.Points += count
}

func (p *Player) ReduceHealth(hp int) {
	p.State.HP -= hp
	if p.State.HP < 0 {
		p.State.HP = 0
	}
}

func (p *Player) CheckZHealth() bool {
	return p.State.HP == 0
}

func (p *Player) CheckNoMobsNoMoney() bool {
	return p.State.AllDead && p.State.Points == 0
}

func GetInitPos(target Target, area Area) Position {
	y := rand.Intn(int(area.Height))
	return Position{
		X: target.Pos.X + target.Area.Width/2 + 25,
		Y: float64(y),
	}
}
