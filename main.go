package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var pathToDatas string

var pgnWaitList = make(chan string)
var pgnBestMoves = make(chan string)

var pgnWaitListSlow = make(chan string)
var pgnBestMovesSlow = make(chan string)

type CmdWrapper struct {
	Cmd      *exec.Cmd
	Pgn      string
	Input    io.WriteCloser
	BestMove chan string
}

func (c *CmdWrapper) openInput() {
	var err error
	c.Input, err = c.Cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
}

var p CmdWrapper
var pSlow CmdWrapper

func (c *CmdWrapper) launch(networkPath string, args []string, input bool, movetime string, pgnWaitListChan chan string, pgnBestMovesChan chan string) {
	c.BestMove = make(chan string)
	weights := fmt.Sprintf("--weights=%s", networkPath)
	c.Cmd = exec.Command("lczero", weights, "-t1")
	c.Cmd.Args = append(c.Cmd.Args, args...)
	//c.Cmd.Args = append(c.Cmd.Args, "--gpu=1")
	c.Cmd.Args = append(c.Cmd.Args, "--quiet")
	c.Cmd.Args = append(c.Cmd.Args, "-n")

	log.Printf("Args: %v\n", c.Cmd.Args)

	stdout, err := c.Cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := c.Cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		reading_pgn := false
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			//log.Printf("%s\n", line)
			if line == "PGN" {
				reading_pgn = true
			} else if line == "END" {
				reading_pgn = false
			} else if reading_pgn {
				c.Pgn += line + "\n"
			} else if strings.HasPrefix(line, "bestmove ") {
				c.BestMove <- strings.Split(line, " ")[1]
			}
		}
	}()

	go func() {
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			log.Printf("%s\n", stderrScanner.Text())
		}
	}()

	if input {
		c.openInput()
	}

	err = c.Cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	io.WriteString(c.Input, "uci\n")
	go func() {
		for pgn := range pgnWaitListChan {
			if len(pgn) > 1 {
				log.Println("position startpos moves " + pgn)
				io.WriteString(p.Input, "position startpos moves "+pgn+" \n")
			} else {
				log.Println("position startpos")
				io.WriteString(p.Input, "position startpos \n")
			}

			log.Println("go movetime " + movetime)
			io.WriteString(p.Input, "go movetime"+movetime+"\n")

			select {
				case best_move := <-p.BestMove:
					pgnBestMovesChan <- best_move
				case <-time.After(10 * time.Second):
					pgnBestMovesChan <- "timeout"
			}
		}
	}()
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	if path == "" {
		path = "index.html"
	}
	page, err := LoadPage(path)

	if err != nil {
		log.Println("Error loading page : ", err)
		w.WriteHeader(404)
		fmt.Fprintf(w, "404 - Page not found !")
	} else {
		log.Println("Page requested and sent : ", page.Title)
		w.Write(page.Body)
	}
}

func getMoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("GET /getMove from", r.RemoteAddr, ": ", r.URL.Query())
		if r.URL.Query().Get("pgn") != "" {
			pgn := r.URL.Query().Get("pgn")
			pgnWaitList <- pgn
			fmt.Fprintf(w, <-pgnBestMoves)
		} else {
			fmt.Fprintf(w, "please provide pgn as uci moves")
		}
	}
}

func getMoveSlowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("GET /getMoveSlow from ", r.RemoteAddr, ": ", r.URL.Query())
		if r.URL.Query().Get("pgn") != "" {
			pgn := r.URL.Query().Get("pgn")
			pgnWaitListSlow <- pgn
			fmt.Fprintf(w, <-pgnBestMovesSlow)
		} else {
			fmt.Fprintf(w, "please provide pgn as uci moves")
		}
	}
}

func main() {
	pathToDatas = "./Data/"
	if len(os.Args) >= 2 {
		logFilePath := os.Args[1]
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err == nil {
			log.SetOutput(f)
		}
	}
	defaultMux := http.NewServeMux()
	defaultMux.HandleFunc("/", defaultHandler)
	defaultMux.HandleFunc("/getMove", getMoveHandler)
	defaultMux.HandleFunc("/getMoveSlow", getMoveSlowHandler)
	p = CmdWrapper{}
	p.launch("networks/3857", nil, true, "200", pgnWaitList, pgnBestMoves)
	pSlow = CmdWrapper{}
	pSlow.launch("networks/3857", nil, true, "2000", pgnWaitListSlow, pgnBestMovesSlow)
	defer p.Input.Close()
	defer pSlow.Input.Close()

	http.ListenAndServe(":80", defaultMux)

}
