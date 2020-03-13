package portforward

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	helloWorld = []byte("hello world!")
)

func TestPortForwardSuite(t *testing.T) {
	suite.Run(t, new(PortForwardSuite))
}

type PortForwardSuite struct {
	suite.Suite
	listenPort  string
	forwardPort string
	stop        chan struct{}
}

func (s *PortForwardSuite) SetupSuite() {
	s.listenPort = fmt.Sprintf(":%d", 1234)
	s.forwardPort = fmt.Sprintf(":%d", 5678)

	// Create a echo tcp server
	go echoTcpServer(s.forwardPort)
}

func (s *PortForwardSuite) TestReconnect() {
	// connection should fail because no one is listen to the port.
	s.writeAndCheck(false)

	// Now start port forwarding
	stop, err := PortForward(s.listenPort, s.forwardPort)
	assert.NoError(s.T(), err)

	// after port forwarding it should work
	s.writeAndCheck(true)
	s.writeAndCheck(true)

	// port forwarding stopped. should fail.
	stop <- struct{}{}
	time.Sleep(1 * time.Second)
	fmt.Println("stopped1")
	s.writeAndCheck(false)

	// port forwarding again. should success.
	stop, err = PortForward(s.listenPort, s.forwardPort)
	s.writeAndCheck(true)
	stop <- struct{}{}
	// time.Sleep(1 * time.Second)
	fmt.Println("stopped2")
}

func (s *PortForwardSuite) writeAndCheck(shouldSuccess bool) {
	conn, err := net.Dial("tcp", s.listenPort)
	if !shouldSuccess {
		assert.Error(s.T(), err)
		return
	}
	assert.NoError(s.T(), err)
	conn.Write(helloWorld)
	buf := make([]byte, 2014)
	l, err := conn.Read(buf)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), helloWorld, buf[:l])
	conn.Close()
}

func echoTcpServer(hostPort string) {
	l, err := net.Listen("tcp", hostPort)
	if err != nil {
		panic(fmt.Sprintf("Error listening:" + err.Error()))
	}

	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + hostPort)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			panic(fmt.Sprintf("Error accepting: %s", err.Error()))
		}

		// Make a buffer to hold incoming data.
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
		}
		conn.Write(buf[:reqLen])
		// Close the connection when you're done with it.
		conn.Close()
	}
}
