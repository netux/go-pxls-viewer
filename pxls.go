package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/websocket"
	"github.com/lucasb-eyer/go-colorful"
)

// PxlsBoard represents a [width][height]color_index pxls board.
type PxlsBoard [][]uint8

// Width returns the length of the PxlsBoard.
func (b *PxlsBoard) Width() int {
	return len(*b)
}

// Height returns the length of the first element of the PxlsBoard, or 0 if there is no first element.
func (b *PxlsBoard) Height() int {
	if len(*b) > 0 {
		return len((*b)[0])
	}

	return 0
}

// Pxls holds data on the current state of the board, and also handles the websocket connection.
type Pxls struct {
	init bool
	conn *websocket.Conn

	Host 			 string
	SecureConn bool

	WsMsgCh 	 chan pxlsMsg
	ErrCh 		 chan error
	Board   	 PxlsBoard
	Palette 	 []termbox.Attribute
}

// NewPxls creates a new Pxls instance.
func NewPxls(host string, secure bool) *Pxls {
	return &Pxls{
		Host: host,
		SecureConn: secure,
		WsMsgCh: make(chan pxlsMsg),
		ErrCh: make(chan error),
	}
}

// IsInit returns whether a connection with the Pxls' servers has been made.
func (pxls *Pxls) IsInit() bool {
	return pxls.init
}

// Init retrieves board data and info and establishes a connection with Pxls' websockets.
func (pxls *Pxls) Init() error {
	if pxls.init {
		return nil
	}

	var apiURLTemplate = "http://%s"
	var wsURLTemplate = "ws://%s/ws"
	if pxls.SecureConn {
		apiURLTemplate = "https://%s"
		wsURLTemplate = "wss://%s/ws"
	}

	var apiURL = fmt.Sprintf(apiURLTemplate, pxls.Host)
	var wsURL = fmt.Sprintf(wsURLTemplate, pxls.Host)

	var err error

	resp, err := http.Get(apiURL + "/info")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	infoBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var info pxlsInfo
	err = json.Unmarshal(infoBytes, &info)
	if err != nil {
		return err
	}

	pxls.Palette = pxls.parsePalette(info.Palette)
	
	resp, err = http.Get(apiURL + "/boarddata")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bd, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pxls.Board = pxls.makeBoard(info.Width, info.Height, bd)

	conn, err := websocket.Dial(wsURL, "", "http://localhost")
	if err != nil {
		return err
	}
	go readPxlsWsLoop(conn, pxls.WsMsgCh, pxls.ErrCh)

	pxls.conn = conn

	pxls.init = true

	return nil
}

func (pxls *Pxls) makeBoard(w, h int, data []uint8) (board PxlsBoard) {
	board = make(PxlsBoard, w)

	for x := range board {
		board[x] = make([]uint8, h)

		for y := range board[x] {
			board[x][y] = data[x + y * w]
		}
	}

	return board
}

func (pxls *Pxls) parsePalette(ip []string) (palette []termbox.Attribute) {
	for _, hex := range ip {
		c, err := colorful.Hex(hex)
		if err != nil {
			panic(err)
		}

		palette = append(palette, colorToTermboxAttr(c))
	}

	return
}

// Close closes the websocket connection.
func (pxls *Pxls) Close() error {
	var err error
	if err = pxls.conn.Close(); err != nil {
		return err
	}

	return nil
}

type pxlsInfo struct {
	Width, Height int
	Palette       []string
}

type pxlsMsg struct {
	Type 		string
	Count   int
	Pixels  []pxlsPixel
}

type pxlsPixel struct {
	X, Y  int
	Color uint8
}

func readPxlsWsLoop(conn *websocket.Conn, msgCh chan pxlsMsg, errCh chan error) {
	for {
		var m pxlsMsg
		err := websocket.JSON.Receive(conn, &m)
		if err != nil {
			errCh <- err
		}

		msgCh <- m
	}
}
