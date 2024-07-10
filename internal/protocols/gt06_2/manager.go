package gt06_2

import (
	"net"
	"strings"
)

type Manager struct {
	Port int
}

func NewManager(port int) *Manager {
	return &Manager{Port: port}
}

func (m *Manager) StartServer() {
	listener, err := net.Listen("tcp", ":"+string(m.Port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go m.handleConnection(conn)
	}
}

func (m *Manager) handleConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}
		bodies := m.bodies(string(buffer[:n]))
		for _, body := range bodies {
			m.handleBody(body, conn)
		}
	}
}

func (m *Manager) bodies(body string) []string {
	return strings.Split(body, "0d0a")
}

func (m *Manager) handleBody(body string, conn net.Conn) {
	// Parse and handle different message types here
	if strings.HasPrefix(body, "7878") {
		authParser := NewAuthParser(body)
		if authParser.IsValid() {
			conn.Write([]byte(authParser.Response()))
		}
	}
	// Add more parsers as needed
}
