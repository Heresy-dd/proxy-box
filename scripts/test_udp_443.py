import socket
import socks
import time

# We have SOCKS5 proxies running on:
# 29200: p-trojan-jp
# 29201: p-trojan-hk
# 29202: p-hy2-hk
# 29203: p-hy2-sg

# Let's start the proxy instance from test_udp if not running
import subprocess
import os

PORT = 29093
SECRET = 'udp123'
WORK_DIR = '/tmp/mihomo_udp'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
cfg_path = os.path.join(WORK_DIR, "config.json")

# Start mihomo if not running
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
time.sleep(1)

def test_socks5_udp(socks_port, target_host, target_port):
    s = socks.socksocket(socket.AF_INET, socket.SOCK_DGRAM)
    s.set_proxy(socks.SOCKS5, "127.0.0.1", socks_port, username="u", password="p")
    s.settimeout(3)
    try:
        t0 = time.time()
        # Send a DNS or QUIC initial packet or dummy packet
        s.sendto(b"\x00\x01\x00\x00", (target_host, target_port))
        # Try to receive (or see if error on send)
        data, addr = s.recvfrom(1024)
        print(f"Port {socks_port} -> {target_host}:{target_port}: Received {len(data)} bytes from {addr} in {int((time.time()-t0)*1000)}ms")
        s.close()
        return True
    except socket.timeout:
        print(f"Port {socks_port} -> {target_host}:{target_port}: Timeout (no response)")
        s.close()
        return False
    except Exception as e:
        print(f"Port {socks_port} -> {target_host}:{target_port}: Error: {e}")
        s.close()
        return False

# Test DNS (UDP 53) vs Target UDP 443
print("=== Testing UDP 53 (DNS 1.1.1.1) through SOCKS5 ===")
for p in [29200, 29201, 29202, 29203]:
    # DNS standard query for example.com
    dns_query = b'\xaa\xbb\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01'
    s = socks.socksocket(socket.AF_INET, socket.SOCK_DGRAM)
    s.set_proxy(socks.SOCKS5, "127.0.0.1", p, username="u", password="p")
    s.settimeout(3)
    try:
        s.sendto(dns_query, ("1.1.1.1", 53))
        data, addr = s.recvfrom(1024)
        print(f"SOCKS5 Port {p} UDP 53 DNS query: SUCCESS ({len(data)} bytes)")
    except Exception as e:
        print(f"SOCKS5 Port {p} UDP 53 DNS query: FAILED ({e})")
    s.close()

print("\n=== Testing UDP 443 to cnix.qianlink.link through SOCKS5 ===")
for p in [29200, 29201, 29202, 29203]:
    s = socks.socksocket(socket.AF_INET, socket.SOCK_DGRAM)
    s.set_proxy(socks.SOCKS5, "127.0.0.1", p, username="u", password="p")
    s.settimeout(3)
    try:
        s.sendto(b"\x00\x01\x00\x00", ("cnix.qianlink.link", 443))
        data, addr = s.recvfrom(1024)
        print(f"SOCKS5 Port {p} UDP 443 to cnix: SUCCESS ({len(data)} bytes)")
    except Exception as e:
        print(f"SOCKS5 Port {p} UDP 443 to cnix: FAILED ({e})")
    s.close()

proc.terminate()
proc.wait()
