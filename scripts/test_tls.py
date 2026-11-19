import socket
import ssl

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(5)
s.connect(("42.194.131.29", 443))
ssl_sock = ctx.wrap_socket(s, server_hostname="vpn.qianlink.link")
cert = ssl_sock.getpeercert(binary_form=True)
print("TLS Handshake SUCCESS!")
print("TLS Version:", ssl_sock.version())
print("Cipher:", ssl_sock.cipher())
ssl_sock.close()
