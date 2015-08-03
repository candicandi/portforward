// Package portward listens a port and forward to another port. It also provides a channel
// that caller can stop forwarding when needed.

package portforward

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
)

// PortForward forward listen port to forward port.
func PortForward(listenHost, forwardHost string) (chan struct{}, error) {
	// Listen to incoming port
	listener, err := net.Listen("tcp", listenHost)
	if err != nil {
		return nil, fmt.Errorf("unable to listen to port %s, err %v", listenHost, err)
	}

	// forwardMap keeps tracks of all opened connection
	forwardMap := make(map[*net.Conn](*net.Conn))
	lock := &sync.Mutex{}
	// port forwarding go routine. It will quit itself upon closing listener.
	go func() {
		for {
			connLis, err := listener.Accept()
			if err != nil {
				fmt.Printf("unable to accept listener %v, err %v", listener, err)
				return
			}
			// fmt.Printf("Accepted connection %v\n", connLis)

			// Dial for forwarding port
			connFor, err := net.Dial("tcp", forwardHost)
			if err != nil {
				fmt.Printf("unable to dial port %s, err %v", forwardHost, err)
				return
			}
			lock.Lock()
			forwardMap[&connLis] = &connFor
			lock.Unlock()
			// fmt.Printf("Connected to localhost %v\n", connFor)

			go func() {
				io.Copy(connLis, connFor)
				connLis.Close()
				connFor.Close()
				lock.Lock()
				delete(forwardMap, &connLis)
				lock.Unlock()
			}()
			go func() {
				io.Copy(connFor, connLis)
				connLis.Close()
				connFor.Close()
				lock.Lock()
				delete(forwardMap, &connLis)
				lock.Unlock()
			}()
		}
	}()

	stop := make(chan struct{})
	go func() {
		log.Info("waiting for stop port forwarding")
		<-stop
		listener.Close()
		lock.Lock()
		for k, v := range forwardMap {
			log.Infof("closing %v, %v", *k, *v)
			(*k).Close()
			(*v).Close()
		}
		lock.Unlock()
		log.Info("Port forwarding closed.")
	}()
	return stop, nil
}
