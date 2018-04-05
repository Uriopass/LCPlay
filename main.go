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

var pgnWaitListUltra = make(chan string)
var pgnBestMovesUltra = make(chan string)

type CmdWrapper struct {
	Cmd      *exec.Cmd
	Pgn      string
	Input    io.WriteCloser
	BestMove chan string
	Winrate chan string
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
var pUltra CmdWrapper

func (c *CmdWrapper) launch(networkPath string, args []string, input bool, playouts string, pgnWaitListChan chan string, pgnBestMovesChan chan string) {
	c.BestMove = make(chan string)
	c.Winrate = make(chan string)
	weights := fmt.Sprintf("--weights=%s", networkPath)
	c.Cmd = exec.Command("lczero", weights, "-t1")
	c.Cmd.Args = append(c.Cmd.Args, args...)
	//c.Cmd.Args = append(c.Cmd.Args, "--gpu=1")
	//c.Cmd.Args = append(c.Cmd.Args, "--quiet")
	c.Cmd.Args = append(c.Cmd.Args, "-n")
	c.Cmd.Args = append(c.Cmd.Args, "--noponder")
	c.Cmd.Args = append(c.Cmd.Args, "-p"+playouts)

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
		last := ""
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			//log.Printf("Playouts: %v said %s\n", playouts, line)
			if line == "PGN" {
				reading_pgn = true
			} else if line == "END" {
				reading_pgn = false
			} else if reading_pgn {
				c.Pgn += line + "\n"
			} else if strings.HasPrefix(line, "bestmove ") {
				c.Winrate <- last
				c.BestMove <- strings.Split(line, " ")[1]
			} else if strings.HasPrefix(line, "info") {
				last = strings.Split(strings.Split(line, "winrate ")[1], " time")[0]
			} else {
				log.Println("Weird line from lczero.exe "+line)
			}
		}
	}()

	go func() {
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			//log.Printf("%s\n", stderrScanner.Text())
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
				io.WriteString(c.Input, "position startpos moves "+pgn+" \n")
			} else {
				log.Println("position startpos")
				io.WriteString(c.Input, "position startpos \n")
			}

			log.Println("go playouts " + playouts)
			io.WriteString(c.Input, "go \n")

			select {
				case winr := <-c.Winrate:
					select {
						case best_move := <-c.BestMove:
							pgnBestMovesChan <- best_move+";"+winr
					}
			}
		}
	}()
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	if path == "" {
		path = "index.html"
		log.Println("For IP ", r.RemoteAddr, "with referer", r.Referer(), "asked index.html!")
	}
	page, err := LoadPage(path)

	if err != nil {
		log.Println("For IP ", r.RemoteAddr, " Error 404: ", err)
		w.WriteHeader(404)
		fmt.Fprintf(w, "404 - Page not found !")
	} else {
		log.Println("For IP ", r.RemoteAddr, ": ", page.Title)
		w.Write(page.Body)
	}
}

func getMoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("GET /getMove from", r.RemoteAddr, ":", r.URL.Query())
		if r.URL.Query().Get("pgn") != "" {
			start := time.Now()
			pgn := r.URL.Query().Get("pgn")
			pgnWaitList <- pgn
			bestMove := <-pgnBestMoves
			fmt.Fprintf(w, bestMove)
			elapsed := time.Since(start)
			log.Println("It took "+fmt.Sprintf("%s", elapsed)+" and answer is "+bestMove)
		} else {
			fmt.Fprintf(w, "please provide pgn as uci moves")
		}
	}
}

func getMoveUltraHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("GET /getMoveUltra from", r.RemoteAddr, ": ", r.URL.Query())
		if r.URL.Query().Get("pgn") != "" {
			start := time.Now()
			pgn := r.URL.Query().Get("pgn")
			pgnWaitListUltra <- pgn
			bestMove := <-pgnBestMovesUltra
			fmt.Fprintf(w, bestMove)
			elapsed := time.Since(start)
			log.Println("It took "+fmt.Sprintf("%s", elapsed)+" and answer is "+bestMove)
		} else {
			fmt.Fprintf(w, "please provide pgn as uci moves")
		}
	}
}

func getMoveSlowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("GET /getMoveSlow from", r.RemoteAddr, ": ", r.URL.Query())
		if r.URL.Query().Get("pgn") != "" {
			start := time.Now()
			pgn := r.URL.Query().Get("pgn")
			pgnWaitListSlow <- pgn
			bestMove := <-pgnBestMovesSlow
			fmt.Fprintf(w, bestMove)
			elapsed := time.Since(start)
			log.Println("It took "+fmt.Sprintf("%s", elapsed)+" and answer is "+bestMove)
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
		defer f.Close()
	}
	net_name := "a8bd"
	defaultMux := http.NewServeMux()
	defaultMux.HandleFunc("/", defaultHandler)
	defaultMux.HandleFunc("/getMove", getMoveHandler)
	defaultMux.HandleFunc("/getMoveSlow", getMoveSlowHandler)
	defaultMux.HandleFunc("/getMoveUltra", getMoveUltraHandler)
	p = CmdWrapper{}
	p.launch("networks/"+net_name, nil, true, "200", pgnWaitList, pgnBestMoves)
	pSlow = CmdWrapper{}
	pSlow.launch("networks/"+net_name, nil, true, "2000", pgnWaitListSlow, pgnBestMovesSlow)
	pUltra = CmdWrapper{}
	pUltra.launch("networks/"+net_name, nil, true, "1", pgnWaitListUltra, pgnBestMovesUltra)
	defer p.Input.Close()
	defer pSlow.Input.Close()
	defer pUltra.Input.Close()

	http.ListenAndServe(":80", defaultMux)

}
