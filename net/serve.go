package net

import "net"

func Serve(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err = handleConn(conn); err != nil {
				conn.Close()
			}
		}()
	}
}

func handleConn(conn net.Conn) error {
	panic("implement me")
}
