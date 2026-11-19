import socket
import struct
import time
import subprocess
import os

PORT = 29093
SECRET = 'udp123'
WORK_DIR = '/tmp/mihomo_udp'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
cfg_path = os.path.join(WORK_DIR, "config.json")

# Start mihomo
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
time.sleep(1)

def socks5_udp_associate(proxy_port, user, pwd):
    # 1. Connect TCP to SOCKS5 server
    s_tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s_tcp.settimeout(3)
    s_tcp.connect(('127.0.0.1', proxy_port))

    # Auth handshake
    s_tcp.sendall(b'\x05\x02\x00\x02') # SOCKS5, 2 methods: No Auth, User/Pass
    res = s_tcp.recv(2)
    if res[1] == 2: # User/Pass
        req = b'\x01' + bytes([len(user)]) + user.encode() + bytes([len(pwd)]) + pwd.encode()
        s_tcp.sendall(req)
        auth_res = s_tcp.recv(2)
        if auth_res[1] != 0:
            raise Exception("Auth failed")

    # UDP ASSOCIATE request: CMD=3
    s_tcp.sendall(b'\x05\x03\x00\x01\x00\x00\x00\x00\x00\x00')
    resp = s_tcp.recv(10)
    if resp[1] != 0:
        raise Exception(f"UDP Associate failed with code {resp[1]}")

    # Parse relay IP and Port
    if resp[3] == 1: # IPv4
        relay_ip = socket.inet_ntoa(resp[4:8])
        relay_port = struct.unpack('!H', resp[8:10])[0]
    else:
        relay_ip = '127.0.0.1'
        relay_port = struct.unpack('!H', resp[-2:])[0]

    return s_tcp, relay_ip, relay_port

def test_udp(proxy_port, target_ip, target_port, payload):
    try:
        s_tcp, relay_ip, relay_port = socks5_udp_associate(proxy_port, "u", "p")
        # Send UDP packet via relay
        s_udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s_udp.settimeout(2.5)

        # SOCKS5 UDP request header: RSV(2) + FRAG(1) + ATYP(1) + DST.ADDR(4) + DST.PORT(2) + Data
        header = b'\x00\x00\x00\x01' + socket.inet_aton(target_ip) + struct.pack('!H', target_port)
        packet = header + payload

        s_udp.sendto(packet, (relay_ip, relay_port))
        data, addr = s_udp.recvfrom(2048)
        s_tcp.close()
        s_udp.close()
        return True, len(data)
    except socket.timeout:
        return False, "timeout"
    except Exception as e:
        return False, str(e)

dns_query = b'\xaa\xbb\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01'
cnix_ip = socket.gethostbyname("cnix.qianlink.link")

print(f"cnix.qianlink.link IP: {cnix_ip}")
for p, name in [(29200, "Trojan-JP"), (29201, "Trojan-HK"), (29202, "Hy2-HK"), (29203, "Hy2-SG")]:
    # Test DNS 1.1.1.1:53
    ok_dns, res_dns = test_udp(p, "1.1.1.1", 53, dns_query)
    # Test cnix 443
    ok_cnix, res_cnix = test_udp(p, cnix_ip, 443, b'\x00\x01\x00\x00')
    print(f"[{name} on port {p}]:")
    print(f"   DNS 1.1.1.1:53  -> ok={ok_dns} res={res_dns}")
    print(f"   CNIX {cnix_ip}:443 -> ok={ok_cnix} res={res_cnix}")

proc.terminate()
proc.wait()
