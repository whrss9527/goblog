# goblog

基于 Go 的 Markdown 博客系统：服务端渲染、Git 仓库为内容存储、可一键部署到 systemd。

## 特性

- **文件存储**：博客内容（文章、分类、标签、页面）以 Markdown 文件形式存放在独立的 Git 仓库中（`blog-data`），运行时按需克隆/拉取，无需数据库。
- **管理后台**：登录后可对文章、页面、项目、分类、标签、阅读清单做增删改查；admin 操作通过 session cookie 鉴权。
  手机上同样可用；编辑器会在浏览器本地自动保留草稿（关页、登录过期、保存失败都不丢稿），保存前校验文章地址，不会覆盖别的文章。
  标签可以改名、合并（改成另一个标签的名字）、删除，文章里的引用自动跟着变；还有文章在用的分类不允许删除。
- **公开前台**：首页、文章页、标签页、分类页、项目、阅读清单、关于页、站内搜索（基于内存索引）。
- **项目展示**：`/projects` 以卡片展示自己做的项目（简介、亮点、技术栈、源码 / 访问地址、介绍文章），
  GitHub 仓库自动显示 Star、主要语言和最近提交；后台粘贴仓库地址即可一键读取，还会列出 GitHub 上最近还没加进来的仓库。
- **RSS / Atom**：启动时生成 `/feed.xml`。
- **Sitemap**：`/sitemap.xml`。
- **评论**：基于 [giscus](https://giscus.app/) GitHub Discussions 评论组件（滚动到附近才加载）。
- **热力图**：每小时定时聚合写入 `heatmap.txt`，用于贡献图展示。
- **优雅退出**：`SIGTERM` 触发，超时时间可配。
- **服务端渲染 Markdown**：文章 HTML 随响应直出（首屏无白屏、无需 jQuery/editor.md、无 JS 也可读、爬虫可见），
  渲染结果与后台 editor.md 预览保持一致；含流程图 / 时序图 / 公式的文章自动回退到浏览器渲染。
- **PWA / 离线阅读**：可安装到桌面与手机主屏；读过的文章自动保存，断网时照常打开，没读过的地址显示离线页并列出可读文章；
  带指纹的静态资源缓存优先。`app.pwa: false` 一键关闭并自动注销已安装的 Service Worker。
- **SEO**：canonical、Open Graph / Twitter Card（自动取文内首图）、JSON-LD（`BlogPosting` / `Blog`）、静态资源长缓存。
- **多端适配**：前台无框架依赖（CSS 变量 + 原生 JS），手机 / 平板 / 桌面自适应，明暗主题跟随系统，
  文章目录在宽屏为粘性侧栏、窄屏为底部抽屉。版本变更见 [CHANGELOG.md](CHANGELOG.md)。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.23+ |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) |
| 模板 | `html/template`（启动时缓存）|
| 配置 | [Viper](https://github.com/spf13/viper) (YAML) |
| 日志 | `log/slog`（自定义 handler，支持 trace id）|
| Markdown | [editor.md](https://github.com/pandao/editor.md)（后台编辑器）+ [blackfriday](https://github.com/russross/blackfriday)（前台服务端渲染，`internal/pkg/md2html`）|
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

### 只想看看效果，不碰线上数据

`data_dir` 指向一个**不含 `.git`** 的目录、并且不配置 `git_repo` 时，goblog 只读写这个目录：不 pull、不 commit、不 push。
拿内容仓库的一份导出当数据，就能放心地点赞、改文章、试后台，线上仓库一个字节都不会变：

```bash
mkdir -p /tmp/goblog-preview/data
git -C /path/to/blog-data archive HEAD | tar -x -C /tmp/goblog-preview/data   # 导出的副本里没有 .git

# conf/preview.yaml：data_dir 指向上面的目录，git_repo 留空，cdn 写 "/static"（不依赖外部 CDN），
# app.host 写 http://localhost:9191，server.host 写 "127.0.0.1"（只有本机能访问），server.http_port 写 9191
./goblog -config ./conf/preview.yaml
```

本机 / 局域网地址（`localhost`、`127.0.0.1`、`192.168.x.x`、`*.local` 等）打开的页面不会加载 Google Analytics，预览不会污染线上统计。

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

## 从 1.0 升级到 1.11

1.1 ～ 1.11 没有破坏性变更：配置文件不改也能启动，内容仓库的文件格式保持不变。照常发版即可：

```bash
cd /opt/goblog && git pull && make build && systemctl restart goblog
```

升级时值得过一遍的事情：

| 事项 | 说明 |
|------|------|
| 静态文件 | `tpl/`、`static/`、`robots.txt` 随仓库更新；用 `make tar` 发版的话包里也已经带上。新增的 `static/icons/`、`static/js/*.js`、`static/admin/` 都从本机 `/static` 加载（带内容指纹），**CDN 桶（`app.cdn`）不需要上传任何新文件** |
| 后台账号 | 内容仓库是公开的话，把账号搬进配置文件并**换一个新密码**，见下文「后台账号放在配置文件里」。没搬之前，后台每个列表页顶部都会有一条提醒 |
| 新配置项 | 全部可选：`description`（站点简介，建议填，首页标题和搜索结果摘要会用）、`markdown_render`、`pwa`、`admin_email` / `admin_password_hash`、`server.host`（用 Cloudflare Tunnel / nginx 时建议 `"127.0.0.1"`）、`github_stats` / `github_user` / `github_token`（项目页），说明见 `conf/prod.yaml.example` |
| robots.txt | 1.9.1 起只屏蔽 `/admin/`、`/api/`、`/random`、`/offline`，并附上 sitemap 地址。此前的内容是 `Disallow: /`（拒绝所有搜索引擎）；如果那是有意的，把仓库根目录的 `robots.txt` 改回去即可 |
| Service Worker | 访客的浏览器会注册 `/sw.js`（离线阅读）。反向代理 / Cloudflare 不要给 `/sw.js` 加长缓存（服务端返回的是 `no-cache`）。想关掉就设 `pwa: false` —— 它会让已安装的 Service Worker 自行注销并清空缓存；**回滚到 1.7 之前的版本，也请先这样跑几天** |
| 内容仓库的新文件 | `likes.json`（点赞数，和 `views.json` 一样每小时提交一次）、`projects.json`（项目，添加第一个项目时生成）。草稿在 `<data_dir>/.drafts/`，不进仓库 |
| 出站网络 | 1.11 起服务器会访问 `api.github.com`（只读公开数据：项目的 Star 等，以及后台导入仓库时）。没有项目就不会请求；不想要可以设 `github_stats: false` |
| 旧标签 | 1.9 之前用「`a, b`」这种写法输入标签，会产生带前导空格的重复标签（如 `" blog"` 和 `"blog"` 并存）。后台「标签」页现在可以改名 / 删除：把带空格的那个**改名成正常的名字**，两个标签就会合并，文章自动换到留下的那个标签下 |

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
  description: ""              # 可选：站点一句话简介，用于首页标题 / meta description / 结构化数据
  markdown_render: server      # 可选：server（默认，服务端直出 HTML）/ client（浏览器内 editor.md 渲染）
  pwa: true                    # 可选：PWA / 离线阅读开关，默认开启；false 会让已安装的 Service Worker 自动注销
  admin_email: ""              # 推荐：后台账号写在配置里（见下文「后台账号放在配置文件里」）
  admin_password_hash: ""      # ./goblog -hash-password 生成
  github_stats: true           # 可选：项目页显示 GitHub 仓库的 Star / 语言 / 最近提交，默认开启
  github_user: ""              # 可选：后台「GitHub 上最近的仓库」列谁的仓库，留空 = git_repo 的所有者
  github_token: ""             # 可选：GitHub API 每小时 60 次 → 5000 次，任何 token 都行，不需要任何权限

server:
  host: ""                     # 可选：监听地址，留空 = 所有网卡；nginx / cloudflared 之后建议 "127.0.0.1"
  http_port: 9091
  graceful_shutdown_timeout: 15s
```

### 后台账号放在配置文件里（推荐）

内容仓库（`blog-data`）里的 `users.json` 保存着后台账号的 bcrypt 密码哈希。**如果内容仓库是公开的，这个哈希任何人都能下载**，
弱密码可以被离线暴力破解。建议：

```bash
./goblog -hash-password        # 输入新密码，得到 $2a$12$... 形式的哈希
```

把邮箱和哈希写进 `conf/prod.yaml`（该文件不进任何仓库）：

```yaml
app:
  admin_email: "you@example.com"
  admin_password_hash: "$2a$12$..."
```

两项都配置后只认这个账号，`users.json` 不再生效；随后可以把 `users.json` 从内容仓库删掉。
旧哈希仍然留在 Git 历史里，所以**一定要换一个新密码**，不要沿用旧的。

### 项目

后台「项目」里添加的项目显示在 `/projects`（有第一个项目之后，导航栏才会出现「项目」），保存在内容仓库的 `projects.json`，
和 `books.json` 一样可以直接手改。

- **从 GitHub 添加**：项目列表页会列出 GitHub 上最近提交过、还没加进来的公开仓库（不含 fork、已归档的仓库），点「添加」自动读取名称、简介、主页、
  语言和话题；也可以在表单里粘贴任意仓库地址（`https://github.com/用户名/仓库名` 或 `用户名/仓库名`）点「从 GitHub 读取」，只填空着的项。
- **实时数据**：GitHub 仓库的 Star、Fork、主要语言、最近一次提交由服务器每 6 小时读取一次（带 ETag 的条件请求，没变化不计入限额），
  只保存在内存里、不写进仓库；GitHub 连不上时页面照常显示，只是没有这些数字。`github_stats: false` 可以关掉。私有仓库和已删除的仓库不会出现源码链接。
- **和文章互相链接**：在项目里选一篇「介绍文章」，项目卡片会链接过去，文章末尾也会出现这个项目。
- 精选的项目排在最前面，并优先出现在首页侧栏的「项目」小组件里；已归档的项目排在最后。

### 草稿

写文章时「存草稿」会把未发布的文章保存在服务器的 `<data_dir>/.drafts/`：不公开、不出现在首页 / 归档 / RSS / sitemap / 搜索里，
也**不会提交到内容仓库**（通过 `.git/info/exclude` 排除，仓库内容本身不被改动）。代价是草稿只存在于这台服务器上 ——
容器部署请把 `data_dir` 放在持久卷里。草稿可以在后台预览，发布时才会创建新标签并进入仓库。
另外，编辑器会在浏览器本地自动备份正在输入的内容（关页、登录过期、保存失败都能恢复）。

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
