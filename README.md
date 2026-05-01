# goblog

基于 Go 的 Markdown 博客系统：服务端渲染、Git 仓库为内容存储、可一键部署到 systemd。

## 特性

- **文件存储**：博客内容（文章、分类、标签、页面）以 Markdown 文件形式存放在独立的 Git 仓库中（`blog-data`），运行时按需克隆/拉取，无需数据库。
- **管理后台**：登录后可对文章、页面、分类、标签做增删改查；admin 操作通过 session cookie 鉴权。
- **公开前台**：首页、文章页、标签页、分类页、阅读清单、关于页、站内搜索（基于内存索引）。
- **RSS / Atom**：启动时生成 `/feed.xml`。
- **Sitemap**：`/sitemap.xml`。
- **评论**：基于 [utterances](https://utteranc.es/) GitHub Issues 评论组件。
- **热力图**：每小时定时聚合写入 `heatmap.txt`，用于贡献图展示。
- **优雅退出**：`SIGTERM` 触发，超时时间可配。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.23+ |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) |
| 模板 | `html/template`（启动时缓存）|
| 配置 | [Viper](https://github.com/spf13/viper) (YAML) |
| 日志 | `log/slog`（自定义 handler，支持 trace id）|
| Markdown | [editor.md](https://github.com/pandao/editor.md)（编辑器）+ Go 端渲染 |
| 内容存储 | 独立 Git 仓库 + 文件系统 |
| 进程管理 | systemd（推荐）|

## 目录结构

```
.
├── Makefile                 # 构建 / 打包 / 格式化
├── Dockerfile               # 可选：容器化部署
├── conf/
│   ├── dev.yaml.example     # 开发配置模板
│   ├── prod.yaml.example    # 生产配置模板
│   └── goblog.service       # systemd 单元模板
├── internal/
│   ├── config/              # 配置加载（viper）
│   ├── filestore/           # 文件存储仓储层
│   ├── handler/
│   │   ├── admin/           # 管理后台路由
│   │   └── front/           # 前台公开路由
│   ├── pkg/                 # 应用内部工具（gin、view、model、slogx）
│   └── routers/             # 路由装配
├── pkg/                     # 通用库（utils、cache、exception）
├── static/                  # 静态资源
├── tpl/                     # 模板（default 前台、admin 后台、intro 介绍页）
├── main.go
└── startup.sh               # 备用：手动启停脚本
```

## 本地运行

```bash
# 1. 克隆仓库
git clone git@github.com:whrss9527/goblog.git
cd goblog

# 2. 准备配置
cp conf/dev.yaml.example conf/dev.yaml
# 至少需要修改：app.git_repo（指向你的 blog-data 仓库），如果是私有仓库还要填 app.git_token
# 用 `openssl rand -hex 32` 生成一个 session_secret 替换占位值
vim conf/dev.yaml

# 3. 编译
make build       # 默认产出 linux/amd64 二进制
make mac         # 或编译为 macOS arm64

# 4. 运行
./goblog -config ./conf/dev.yaml
# 浏览器打开 http://localhost:9091
```

## 生产部署（systemd）

```bash
# 服务器上
git clone https://github.com/whrss9527/goblog.git /opt/goblog
cd /opt/goblog

# 配置（参考 conf/prod.yaml.example，注意改 host 字段为对外域名）
vim conf/prod.yaml

# 构建（服务器需安装 Go 1.23+）
make build

# 安装 systemd 服务
mkdir -p /var/lib/goblog/data /var/log/goblog
cp conf/goblog.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now goblog
systemctl status goblog
```

更新发版：
```bash
cd /opt/goblog && git pull && make build && systemctl restart goblog
```

## 配置说明

最小化配置（见 `conf/dev.yaml.example`）：

```yaml
app:
  name: "你的博客名"
  mode: release                # debug / release
  host: https://your-domain    # 对外可访问的 URL，sitemap 和文章绝对链接会用
  session_secret: "<32 字节随机串>"
  data_dir: "/var/lib/goblog/data"
  git_repo: "https://github.com/your-username/blog-data.git"
  git_token: ""                # 私有 blog-data 仓库才需要填 PAT (Contents: Read)

server:
  http_port: 9091
  graceful_shutdown_timeout: 15s
```

## 开发规范

遵循 [Uber Go 编码规范](https://github.com/uber-go/guide/blob/master/style.md)。

提交前：
```bash
make fmt   # gofmt -s -w .
go test ./...
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make build` | 交叉编译 linux/amd64 |
| `make mac` | 编译 macOS arm64 |
| `make tidy` | `go mod tidy` |
| `make fmt` | 格式化代码 |
| `make tar` | 打成发布 tar.gz |
| `make clean` | 清理产物 |

## License

MIT
