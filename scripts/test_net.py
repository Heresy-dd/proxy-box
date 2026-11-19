import socket
import ssl
import time

# 1. Test DNS
try:
    ip = socket.gethostbyname("cnix.qianlink.link")
    print(f"DNS cnix.qianlink.link -> {ip}")
except Exception as e:
    print(f"DNS failed: {e}")
    ip = "42.194.131.29"

# 2. Test TCP to 443
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(3)
try:
    s.connect((ip, 443))
    print(f"TCP 443 connect to {ip}: SUCCESS")
    s.close()
except Exception as e:
    print(f"TCP 443 connect to {ip}: FAILED ({e})")

# 3. Test UDP socket
s_udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s_udp.settimeout(3)
try:
    # Send a dummy packet
    s_udp.sendto(b"\x00\x00\x00\x00", (ip, 443))
    print(f"UDP 443 sendto {ip}: SUCCESS (packet sent)")
except Exception as e:
    print(f"UDP 443 sendto {ip}: FAILED ({e})")
s_udp.close()
