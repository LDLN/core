package main

import (
	"log"
	"github.com/tarm/serial"
)

func main() {
	
	c := &serial.Config{Name: "/dev/ttyS0", Baud: 38400}
	
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	
	n, err := s.Write([]byte("LDLN Serial is listening"))
	if err != nil {
		log.Fatal(err)
	}

	for {
        buf := make([]byte, 128)
        n, err = s.Read(buf)
        if err != nil {
                log.Fatal(err)
        }
        log.Printf("%q", buf[:n])
	}
}