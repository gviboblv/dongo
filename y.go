package main

import (
	"fmt"
	"net"
)

// BAHAYA! Jangan pernah deploy ini ke production!
const (
	PORT = ":123"
	// Amplification yang sangat berbahaya
	RESPONSE_SIZE = 1024 * 1024 * 2 // 2MB response
)

func main() {
	addr, _ := net.ResolveUDPAddr("udp", PORT)
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	fmt.Printf("DANGEROUS amplifier running on %s\n", PORT)
	fmt.Printf("Amplification factor: %dx\n", RESPONSE_SIZE/512)
	
	buffer := make([]byte, 512)
	
	// Generate massive response payload
	response := make([]byte, RESPONSE_SIZE)
	for i := range response {
		response[i] = 0x1c
	}

	fmt.Printf("WARNING: Each 512-byte request will generate %d MB response!\n", RESPONSE_SIZE/(1024*1024))
	fmt.Printf("Potential bandwidth amplification: %.0fx\n", float64(RESPONSE_SIZE)/512.0)

	requestCount := 0
	totalBytesSent := uint64(0)

	for {
		_, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}
		
		// Send massive response
		conn.WriteToUDP(response, remoteAddr)
		
		requestCount++
		totalBytesSent += uint64(RESPONSE_SIZE)
		
		if requestCount%100 == 0 {
			fmt.Printf("Processed %d requests, sent %.2f GB total\n", 
				requestCount, float64(totalBytesSent)/(1024*1024*1024))
		}
	}
}

// Calculation functions for impact analysis
func calculateBandwidthAmplification(inputSize, outputSize int) float64 {
	return float64(outputSize) / float64(inputSize)
}

func calculateCostImpact(gbpsOutput float64, durationHours float64, costPerGB float64) float64 {
	totalGB := gbpsOutput * 8 * 3600 * durationHours / 8 // Convert to GB
	return totalGB * costPerGB
}

func demonstrateImpact() {
	fmt.Println("\n=== IMPACT ANALYSIS ===")
	
	inputSize := 512
	outputSize := 2 * 1024 * 1024 // 2MB
	
	amplification := calculateBandwidthAmplification(inputSize, outputSize)
	fmt.Printf("Amplification factor: %.0fx\n", amplification)
	
	// Scenario: Attacker with 10 Mbps connection
	attackerBandwidth := 10.0 // Mbps
	amplifiedBandwidth := attackerBandwidth * amplification / 8 // Convert to Gbps
	
	fmt.Printf("Attacker bandwidth: %.1f Mbps\n", attackerBandwidth)
	fmt.Printf("Amplified attack bandwidth: %.1f Gbps\n", amplifiedBandwidth)
	
	// Cost calculation (AWS pricing example)
	awsCostPerGB := 0.09
	costPer1Hour := calculateCostImpact(amplifiedBandwidth, 1.0, awsCostPerGB)
	costPer24Hours := calculateCostImpact(amplifiedBandwidth, 24.0, awsCostPerGB)
	
	fmt.Printf("\nPotential cost impact (AWS bandwidth pricing):\n")
	fmt.Printf("1 hour attack: $%.2f\n", costPer1Hour)
	fmt.Printf("24 hour attack: $%.2f\n", costPer24Hours)
	
	fmt.Println("\n=== MITIGATION STRATEGIES ===")
	fmt.Println("1. Rate limiting per IP")
	fmt.Println("2. Response size limits")
	fmt.Println("3. Source IP validation")
	fmt.Println("4. DDoS protection services")
	fmt.Println("5. Network-level filtering")
}