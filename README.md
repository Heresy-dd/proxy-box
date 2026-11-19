# ProxyBox —— 基于 Mihomo 的链式代理与端口映射管理面板

## 1. 项目背景与流量拓扑
在多环境指纹浏览器使用场景中，通常需要将高速中继机场（前置节点）与纯净住宅 IP（落地节点）串联为链路，并为每个浏览器指纹环境提供独立的本地监听端口：
```
指纹浏览器环境
      ↓ (HTTP / SOCKS5 代理请求，带用户名密码鉴权)
[ProxyBox 服务器: 9001 ~ 9005] (Mihomo Mixed 监听器)
      ↓ (dialer-proxy 链式中继)
[前置代理节点] (机场高速专线 / 直连)
      ↓ (公网隧道)
[住宅落地节点] (海外家宽住宅 IP)
      ↓
[目标网站] (看到纯净住宅 IP 出口)
```

---

## 2. 核心功能与特性

### 1. 订阅合并与智能清洗
- **多订阅源纳管**：支持同时添加机场订阅与住宅代理订阅，支持手动即时刷新与定时自动拉取。
- **容灾缓存**：拉取失败或源站异常时自动保留上一版本节点数据，界面高亮报错预警，防止业务链路意外中断。
- **协议自适应**：优先解析 Clash/Mihomo YAML；自动降级 Base64 逐行协议解析（支持 SS / VMess / VLESS / Trojan / Hysteria2 / TUIC）。
- **流量与伪节点过滤**：精准提取 `Subscription-Userinfo` 响应头（显示已用/总流量与到期时间）；自动过滤「剩余流量、官网、套餐、群通知」等 20+ 关键词信息类非代理节点。
- **稳定 Node ID**：采用 `SHA256(订阅ID + ":" + 原始节点名 + ":" + 服务器 + ":" + 端口)` 确定性算法。订阅刷新后，三池归类、排序、置顶和连线均完整保留。

### 2. 顶部三池分类与拖拽排序
- **三池架构**：`未归类池` | `前置池 (复用节点)` | `住宅池 (落地节点)`。
- **多选与批量归类**：支持点击复选框、Ctrl/Shift 连续选择，一键批量在池之间转移。
- **池内持久化排序**：支持排序上下移动与拖拽，常用节点置顶保存，全局下拉框优先选用池内自定义顺序。
- **协议与地区快捷过滤**：按协议（SS/Trojan/VLESS/HY2/TUIC）及地区（🇭🇰 香港、🇸🇬 新加坡、🇯🇵 日本、🇺🇸 美国、🇹🇼 台湾）快速检索。

### 3. 前置复用与链式延迟流式测速
- **一对多复用**：同一个前置中继节点可无限制关联多个不同住宅落地端口。
- **双 Mihomo 实例隔离**：
  - **生产实例（Production）**：监听 `127.0.0.1:9090`，负责业务端口流量（9001+），带用户名密码认证。
  - **探测实例（Probe）**：监听 `127.0.0.1:19090`，不开放对外代理端口，仅在测速时动态克隆生成测试拓扑。**测速全程不触发生产实例热重载，对运行中链路零抖动**。
- **SSE 实时流式推送**：点击前置节点后，住宅池即刻并发测试「前置 → 落地」端到端链式延迟（<300ms 绿、<800ms 黄、更高红、超时灰），测完一个实时推送显示一个。

### 4. 画布拓扑与端口映射
- **两列拓扑画布**：左列为前置中继卡片，右列为住宅落地与端口卡片。
- **自适应端口分配**：默认从 `9001` 递增探测并自动分配下一个未占用端口；双重检测（面板内部占用检测 + 服务器操作系统真实 TCP 绑定检测）。
- **指纹浏览器一键复制**：
  - `SOCKS5`: `socks5://user:pass@服务器IP:端口`
  - `HTTP`: `http://user:pass@服务器IP:端口`
- **出口 IP 核验**：一键经由本机该端口发起外网请求（`api.ipify.org` / `ip-api.com`），直接校验指纹浏览器所见的真实出口 IP 与地理位置。
- **安全白名单与回滚**：变更前显示 Diff 摘要（端口增减明细）；`mihomo -t` 预校验；热重载失败自动回滚上一可用版本，并保留最近 20 个配置版本。

---

## 3. 设计决策与协议适配说明

1. **为什么选择 Go + SQLite + React 单二进制方案？**
   - 数据库选用纯 Go 实现的 `modernc.org/sqlite`（无 CGO 依赖），配合 Go 标准库 `//go:embed` 将前端构建产物 `dist/` 直接打入单一独立可执行文件，服务器部署零外部依赖，极易维护与备份。
2. **为什么测速需要独立的探测实例？**
   - 若在生产实例中临时注入测速节点，每次测速都需要执行 `PUT /configs` 重新加载整个代理池，会导致现有浏览器正在进行的 HTTPS/SOCKS5 连接被重置断开。采用探测实例可在完全隔离的环境下完成任意组合的链式延迟评估。
3. **协议链式代理（dialer-proxy）避坑提示**：
   - **XTLS Vision 限制**：VLESS 的 `flow: xtls-rprx-vision` 专为直接浏览器 TLS 连接设计，若被其他代理嵌套作为 dialer-proxy 会破坏握手。ProxyBox 在生成链路配置时会自动校验并提供警告提示。
   - **UDP 落地依赖**：TUIC 与 Hysteria2 属于基于 QUIC/UDP 的协议。若落地为 TUIC/Hysteria2，前置中继节点必须支持完整的 UDP 转发能力。

---

## 4. 安装与部署指南

### 环境需求
- Linux 服务器（x86_64 或 aarch64，Debian 11+ / Ubuntu 20.04+ / CentOS 8+ 等）
- 建议内存 ≥ 1GB，预留 9001~9999 端口池

### 方式一：一键脚本自动化安装（推荐）
在服务器终端中执行：
```bash
sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/.../install.sh || cat /data/proxybox/scripts/install.sh)"
```
或者进入项目目录直接执行：
```bash
chmod +x ./scripts/install.sh
sudo ./scripts/install.sh
```
安装脚本将自动：
1. 检测当前服务器 CPU 架构（amd64 / arm64）。
2. 下载匹配的最新 Mihomo Meta 内核二进制至 `/data/proxybox/bin/mihomo`。
3. 部署并启动 systemd 系统服务 `proxybox.service`。

### 方式二：手动编译启动
```bash
# 1. 编译前端
cd web && npm install && npm run build && cd ..

# 2. 编译单一二进制
go build -o proxybox ./cmd/server

# 3. 运行服务
./proxybox -addr :8080 -data ./data -mihomo /usr/local/bin/mihomo
```

---

## 5. 服务运维、升级与备份

### 服务管理命令
```bash
# 查看面板运行状态
systemctl status proxybox

# 重启面板与代理服务
systemctl restart proxybox

# 查看面板实时日志
journalctl -u proxybox -f -n 100
```

### 数据备份与迁移
ProxyBox 所有状态均存储在单文件 SQLite 与配置文件中，备份仅需备份数据目录：
```bash
# 备份数据
tar -czvf proxybox_backup_$(date +%Y%m%d).tar.gz /data/proxybox/data

# 恢复数据
tar -xzvf proxybox_backup_20261119.tar.gz -C /
systemctl restart proxybox
```

### 防火墙放行端口
指纹浏览器需要连接面板及分配的代理端口（默认面板 8080，代理端口 9001~9020）：
```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 8080/tcp
sudo ufw allow 9001:9050/tcp

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=9001-9050/tcp
sudo firewall-cmd --reload

# 安全建议：若有固定办公 IP，建议在面板「系统设置 - 来源 IP 白名单」中配置允许连接的客户端 IP。
```
