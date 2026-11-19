#!/usr/bin/env bash
set -e

# ProxyBox One-Click Installer for Linux (systemd)
# Supports Debian/Ubuntu, CentOS/RHEL, Alpine (systemd)

INSTALL_DIR=${INSTALL_DIR:-"/data/proxybox"}
PANEL_PORT=${PANEL_PORT:-8080}
MIHOMO_VERSION=${MIHOMO_VERSION:-"v1.19.34"}

echo "=================================================="
echo "    ProxyBox 链式代理管理面板 一键安装脚本"
echo "=================================================="

# 1. Check root or sudo
if [ "$(id -u)" -ne 0 ]; then
    echo "[-] 提示：本安装脚本需要 root 或 sudo 权限以配置 systemd 服务。"
    exit 1
fi

# 2. Detect CPU architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        MIHOMO_ARCH="linux-amd64"
        ;;
    aarch64|arm64)
        MIHOMO_ARCH="linux-arm64"
        ;;
    *)
        echo "[-] 不支持的 CPU 架构: $ARCH"
        exit 1
        ;;
esac

echo "[+] 目标安装目录: $INSTALL_DIR"
echo "[+] 架构检测: $ARCH ($MIHOMO_ARCH)"
echo "[+] 面板端口: $PANEL_PORT"

mkdir -p "$INSTALL_DIR/bin"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/mihomo_prod"
mkdir -p "$INSTALL_DIR/mihomo_probe"

# 3. Download Mihomo binary
MIHOMO_BIN="$INSTALL_DIR/bin/mihomo"
if [ ! -f "$MIHOMO_BIN" ]; then
    echo "[+] 正在下载 Mihomo Meta 内核 ($MIHOMO_VERSION)..."
    DOWNLOAD_URL="https://github.com/MetaCubeX/mihomo/releases/download/${MIHOMO_VERSION}/mihomo-${MIHOMO_ARCH}-${MIHOMO_VERSION}.gz"
    curl -sL "$DOWNLOAD_URL" -o /tmp/mihomo.gz
    gzip -d -f /tmp/mihomo.gz
    mv /tmp/mihomo "$MIHOMO_BIN"
    chmod +x "$MIHOMO_BIN"
    echo "[+] Mihomo 安装成功: $($MIHOMO_BIN -v)"
else
    echo "[+] Mihomo 二进制已存在: $($MIHOMO_BIN -v)"
fi

# 4. Check or build ProxyBox binary
PROXYBOX_BIN="$INSTALL_DIR/proxybox"
if [ -f "./proxybox" ]; then
    cp ./proxybox "$PROXYBOX_BIN"
    chmod +x "$PROXYBOX_BIN"
elif command -v go >/dev/null 2>&1; then
    echo "[+] 正在编译 ProxyBox 二进制..."
    go build -o "$PROXYBOX_BIN" ./cmd/server
    chmod +x "$PROXYBOX_BIN"
    echo "[+] 编译成功!"
else
    echo "[-] 未找到已编译的 proxybox 二进制且未检测到 Go 编译环境。"
    exit 1
fi

# 5. Create systemd service
SERVICE_FILE="/etc/systemd/system/proxybox.service"
echo "[+] 创建 systemd 服务: $SERVICE_FILE"

cat << EOF > "$SERVICE_FILE"
[Unit]
Description=ProxyBox Mihomo Chained Proxy Management Panel
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$PROXYBOX_BIN -addr :$PANEL_PORT -data $INSTALL_DIR/data -mihomo $MIHOMO_BIN -prod-port 9090 -probe-port 19090
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable proxybox
systemctl restart proxybox

echo ""
echo "=================================================="
echo "    恭喜！ProxyBox 服务已成功安装并启动！"
echo "=================================================="
echo "  Web 面板访问地址: http://<服务器IP>:$PANEL_PORT"
echo "  数据与数据库目录: $INSTALL_DIR/data"
echo "  生产实例配置目录: $INSTALL_DIR/mihomo_prod"
echo "  服务管理命令: "
echo "    - 查看状态: systemctl status proxybox"
echo "    - 重启服务: systemctl restart proxybox"
echo "    - 查看日志: journalctl -u proxybox -f"
echo "=================================================="
