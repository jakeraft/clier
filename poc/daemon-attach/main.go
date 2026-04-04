package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

const (
	sockDir  = "/tmp/clier-poc"
	sockPath = sockDir + "/daemon.sock"

	cmdAttach byte = 'a'
	cmdStop   byte = 's'

	detachByte byte = 0x1c // Ctrl+\

	ringSize = 256 * 1024 // 256KB output history
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: clier-poc <start|attach|stop> [command...]")
		fmt.Fprintln(os.Stderr, "  start [cmd]  start daemon (default: claude)")
		fmt.Fprintln(os.Stderr, "  attach       attach to running session (Ctrl+] to detach)")
		fmt.Fprintln(os.Stderr, "  stop         stop daemon and agent")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "start":
		cmdStart()
	case "attach":
		cmdAttachFn()
	case "stop":
		cmdStopFn()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// ═══════════════════════════════════════════════════════
// START — fork a background daemon that owns the PTY
// ═══════════════════════════════════════════════════════

func cmdStart() {
	// Already running?
	if conn, err := net.Dial("unix", sockPath); err == nil {
		conn.Close()
		die("daemon already running — attach or stop first")
	}

	// Forked child runs the actual daemon
	if os.Getenv("_CLIER_POC_DAEMON") == "1" {
		runDaemon()
		return
	}

	// Parent: fork and exit
	_ = os.MkdirAll(sockDir, 0o755)

	logFile, err := os.Create(sockDir + "/daemon.log")
	if err != nil {
		die("log: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		die("executable: %v", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "_CLIER_POC_DAEMON=1")
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		die("fork: %v", err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Release()
	logFile.Close()

	// Wait for socket to become connectable
	for i := 0; i < 50; i++ {
		if c, err := net.Dial("unix", sockPath); err == nil {
			c.Close()
			fmt.Printf("daemon started (pid %d)\n", pid)
			fmt.Println("  attach:  clier-poc attach")
			fmt.Println("  detach:  Ctrl+\\")
			fmt.Println("  stop:    clier-poc stop")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	die("daemon failed to start — check %s/daemon.log", sockDir)
}

// runDaemon is executed inside the forked child.
func runDaemon() {
	os.Remove(sockPath)

	// Determine which command to run
	args := os.Args[2:] // everything after "start"
	if len(args) == 0 {
		args = []string{"claude"}
	}

	// Spawn agent with PTY
	agentCmd := exec.Command(args[0], args[1:]...)
	ptmx, err := pty.Start(agentCmd)
	if err != nil {
		die("pty: %v", err)
	}
	defer ptmx.Close()

	ring := newRingBuf(ringSize)

	var mu sync.Mutex
	var curClient net.Conn

	// PTY output → ring buffer + current client
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			chunk := buf[:n]
			ring.write(chunk)

			mu.Lock()
			if curClient != nil {
				curClient.Write(chunk)
			}
			mu.Unlock()
		}
	}()

	// Listen on Unix socket
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		die("listen: %v", err)
	}
	defer ln.Close()
	defer os.Remove(sockPath)

	stopCh := make(chan struct{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			var hdr [1]byte
			if _, err := io.ReadFull(conn, hdr[:]); err != nil {
				conn.Close()
				continue
			}
			switch hdr[0] {
			case cmdAttach:
				handleAttach(conn, ptmx, ring, &mu, &curClient)
			case cmdStop:
				conn.Write([]byte("stopped\n"))
				conn.Close()
				select {
				case <-stopCh:
				default:
					close(stopCh)
				}
			default:
				conn.Close()
			}
		}
	}()

	// Wait for agent exit, signal, or stop command
	procDone := make(chan struct{})
	go func() { agentCmd.Wait(); close(procDone) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-procDone:
	case <-sigCh:
		agentCmd.Process.Signal(syscall.SIGTERM)
	case <-stopCh:
		agentCmd.Process.Signal(syscall.SIGTERM)
	}
}

func handleAttach(conn net.Conn, ptmx *os.File, ring *ringBuf, mu *sync.Mutex, cur *net.Conn) {
	// Read client terminal size: rows(2 BE) + cols(2 BE)
	var sz [4]byte
	if _, err := io.ReadFull(conn, sz[:]); err != nil {
		conn.Close()
		return
	}
	rows := binary.BigEndian.Uint16(sz[0:2])
	cols := binary.BigEndian.Uint16(sz[2:4])
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: rows, Cols: cols})

	// Kick previous client (single-attach)
	mu.Lock()
	if *cur != nil {
		(*cur).Close()
	}
	*cur = conn
	mu.Unlock()

	// Flush buffered history so attach feels instant
	if hist := ring.bytes(); len(hist) > 0 {
		conn.Write(hist)
	}

	// Client stdin → PTY (goroutine; cleans up on disconnect)
	go func() {
		defer func() {
			mu.Lock()
			if *cur == conn {
				*cur = nil
			}
			mu.Unlock()
			conn.Close()
		}()
		io.Copy(ptmx, conn)
	}()
}

// ═══════════════════════════════════════════════════════
// ATTACH — connect to daemon, proxy terminal I/O
// ═══════════════════════════════════════════════════════

func cmdAttachFn() {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		die("not running — start first")
	}
	defer conn.Close()

	// Send attach command + terminal size
	conn.Write([]byte{cmdAttach})

	cols, rows, _ := term.GetSize(int(os.Stdin.Fd()))
	if cols == 0 {
		cols, rows = 80, 24
	}
	var sz [4]byte
	binary.BigEndian.PutUint16(sz[0:2], uint16(rows))
	binary.BigEndian.PutUint16(sz[2:4], uint16(cols))
	conn.Write(sz[:])

	// Raw mode — all keystrokes go to the agent, not the local shell
	old, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		die("raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), old)

	// Daemon output → user's stdout
	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn)
		close(done)
	}()

	// User's stdin → daemon (Ctrl+] detaches)
	go func() {
		var buf [1]byte
		for {
			if _, err := os.Stdin.Read(buf[:]); err != nil {
				conn.Close()
				return
			}
			if buf[0] == detachByte {
				conn.Close()
				return
			}
			conn.Write(buf[:])
		}
	}()

	<-done

	// Restore terminal before printing message
	term.Restore(int(os.Stdin.Fd()), old)
	fmt.Println("\ndetached — session still running (attach again or stop)")
}

// ═══════════════════════════════════════════════════════
// STOP — tell daemon to shut down
// ═══════════════════════════════════════════════════════

func cmdStopFn() {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		die("not running")
	}
	defer conn.Close()
	conn.Write([]byte{cmdStop})
	io.Copy(os.Stdout, conn)
}

// ═══════════════════════════════════════════════════════
// Ring buffer — keeps recent output for late-joining attach
// ═══════════════════════════════════════════════════════

type ringBuf struct {
	mu   sync.Mutex
	data []byte
	cap  int
	pos  int
	full bool
}

func newRingBuf(cap int) *ringBuf {
	return &ringBuf{data: make([]byte, cap), cap: cap}
}

func (r *ringBuf) write(p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range p {
		r.data[r.pos] = b
		r.pos++
		if r.pos >= r.cap {
			r.pos = 0
			r.full = true
		}
	}
}

func (r *ringBuf) bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return append([]byte{}, r.data[:r.pos]...)
	}
	out := make([]byte, r.cap)
	n := copy(out, r.data[r.pos:])
	copy(out[n:], r.data[:r.pos])
	return out
}

// ═══════════════════════════════════════════════════════

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
