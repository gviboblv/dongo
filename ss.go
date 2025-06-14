package main
///otak
import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
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
	numWorkers   = 64          // Increased for EPYC 9754
	totalPackets = 1000000     // Increased for stress testing
	batchSize    = 500         // Larger batch size
	bufferSize   = 2048        // Buffered channel size
)

var (
	packetData    []byte
	packetsSent   uint64
	packetsQueued uint64
	shouldStop    atomic.Bool
	wg            sync.WaitGroup
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

func worker(id int, handle *pcap.Handle, jobs <-chan struct{}) {
	defer wg.Done()
	
	// Pre-allocate batch buffer for this worker
	localBatch := make([]byte, len(packetData)*batchSize)
	for i := 0; i < batchSize; i++ {
		copy(localBatch[i*len(packetData):], packetData)
	}

	for range jobs {
		if shouldStop.Load() {
			return
		}
		
		// Send entire batch
		for i := 0; i < batchSize; i++ {
			if shouldStop.Load() {
				return
			}
			
			start := i * len(packetData)
			end := (i + 1) * len(packetData)
			
			err := handle.WritePacketData(localBatch[start:end])
			if err != nil {
				continue
			}
			atomic.AddUint64(&packetsSent, 1)
		}
	}
}

func setupSignalHandling(handle *pcap.Handle) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		cleanup(handle)
	}()
}

func cleanup(handle *pcap.Handle) {
	shouldStop.Store(true)
	wg.Wait()
	handle.Close()
	fmt.Printf("\r\033[K")
	os.Exit(0)
}

func startMonitoring() chan bool {
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
				queuedCount := atomic.LoadUint64(&packetsQueued)
				pps := currentCount - lastCount
				
				fmt.Printf("\rcurrent pps: %d | total sent: %d | queued: %d", 
					pps, currentCount, queuedCount)
				lastCount = currentCount
			case <-stopMonitor:
				return
			}
		}
	}()
	
	return stopMonitor
}

func main() {
	// Optimize for multi-core performance
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	fmt.Printf("starting udp packet sender\n")
	fmt.Printf("using %d cpu cores\n", runtime.NumCPU())
	fmt.Printf("target packets: %d\n", totalPackets)
	fmt.Printf("workers: %d\n", numWorkers)

	handle, err := pcap.OpenLive(iface, 65536, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	setupSignalHandling(handle)
	
	packetData = buildPacket()
	numBatches := totalPackets / batchSize
	jobs := make(chan struct{}, bufferSize)

	// Start worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(w, handle, jobs)
	}

	stopMonitor := startMonitoring()
	start := time.Now()

	// Queue jobs
	go func() {
		for j := 0; j < numBatches; j++ {
			if shouldStop.Load() {
				break
			}
			jobs <- struct{}{}
			atomic.AddUint64(&packetsQueued, batchSize)
		}
		close(jobs)
	}()

	wg.Wait()
	close(stopMonitor)

	if !shouldStop.Load() {
		duration := time.Since(start)
		finalCount := atomic.LoadUint64(&packetsSent)
		avgPPS := float64(finalCount) / duration.Seconds()
		
		fmt.Printf("\ncompleted: %d packets in %v\n", finalCount, duration)
		fmt.Printf("average pps: %.2f\n", avgPPS)
		fmt.Printf("peak throughput: %.2f mbps\n", (avgPPS*float64(len(packetData))*8)/1_000_000)
	}

	cleanup(handle)
}