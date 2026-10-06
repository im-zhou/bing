# 必应壁纸 · Go 版

将原 Node.js/Express 项目迁移为 Go，保留原有网站、MySQL 表和主要接口。原作：xCss / 云淡清风；历史修改：skyxingcheng / IM落蝶花尽。迁移基于 im-zhou/bing 的提交 `0737d8d`。

## 数据和存储

- **无需修改数据库结构，无需迁移历史数据。** Go 不执行 AutoMigrate、ALTER、DROP 或 TRUNCATE。
- 原有 `bing`、`bing_session` 两张表保留；已有库直接填写连接信息。
- 新同步只请求 Bing 图片档案 JSON，将图片标识、日期、标题、版权和链接写入 `bing`。不下载归档图片，不上传七牛/OSS，不保存图片二进制。
- `qiniu_url` 保留为兼容旧详情链接的标识字段，新记录存储 `OHR.*` 图片标识；它已不代表七牛地址。
- 图片地址优先从 `urlbase` 提取，其次使用 `qiniu_url`、`fetch_url` 或 `url` 中的 Bing `id`。历史 `url` 中的旧 CDN 地址无需 UPDATE，新服务不会请求它。
- 新记录 `url`、`fetch_url` 保存 Bing 原图地址。页面直接加载 Bing，JSON 接口将旧 `thumbnail_pic`、`bmiddle_pic`、`original_pic` 响应字段重新生成成 Bing 地址，不需要覆盖旧数据库字段。
- 字段缺失或 NULL 会按空字符串/0 读取；不存在图片标识的坏记录无法恢复图片。
- **不要对已有数据库重新导入 `bing.sql`！** 原文件包含 DROP TABLE；仅用于全新空库初始化。该文件原样保留。

## 构建和启动

需要 Go 1.22+，MySQL/MariaDB，以及能访问 Bing 的网络。

```sh
cp config.example.yaml config.yaml
# 编辑 config.yaml 中的 MySQL 信息、root_url、监听地址
# 若使用已提供的 Linux 二进制，可跳过下面两个命令
go mod download
go build -trimpath -ldflags='-s -w' -o bing .
./bing -config config.yaml
```

静态资源、HTML 模板和时区数据库通过 embed 编入二进制。部署时仅需 `bing` 和 `config.yaml`，无需 Node.js、npm 或外置 static 目录。

配置示例见 `config.example.yaml`；密码也可通过环境变量 `BING_MYSQL_PASSWORD` 覆盖。配置文件保存在程序外，默认被 Git 忽略。默认端口保持 `12345`，默认语言 `zh-CN`，时区 `Asia/Shanghai`，每天 01:00 同步。

启动会同步最近 8 条可用记录，用于恢复短时间停机遗漏；Bing 档案接口并非任意年份的历史目录，长时间缺失仍需要已有数据库提供历史图片元数据。同步失败会最多重试 3 次。用 `sync_on_start:false` 关闭启动同步，`sync_cron:""` 关闭定时同步。

配置采用 YAML，支持注释并拒绝未知字段、重复字段和多个文档。旧 JSON 配置的字段名称保持不变，可按 `config.example.yaml` 转写；默认配置文件改为 `config.yaml`。

首页顶部仅一个搜索框，输入后按回车提交。统一匹配日期、标题、版权描述和文件名/图片标识；支持 `20261006`、`2026-10-06`、`2026/10/06`、`2026年10月6日`，也支持按年/月检索。清空搜索框后回车恢复全部图片。分页和详情返回保留搜索条件。搜索使用原表字段，不新增字段或索引。

只同步一次后退出：

```sh
./bing -config config.yaml -sync-once
```

原表是 MyISAM，并没有 `(enddate,mkt)` 唯一索引。Go 使用绑定独立连接的 MySQL GET_LOCK 和日期/语言检查，在多个 Go 实例间避免重复插入；不借用无效的 MyISAM 事务。不清理已存在的重复记录。切换时应停止旧 Node 服务，旧代码不会遵守新同步锁。

## 接口

| 路径 | 方法 | 功能 |
| --- | --- | --- |
| `/` | GET | 首页，`p` 分页，每页 12 张，`q` 搜索日期/标题/文件名 |
| `/ranking` | GET | 下载榜，`p` 分页，`q` 搜索 |
| `/photo/{qiniu_url}` | GET | 保留旧图片详情链接；无标识的记录可通过数字 ID 访问 |
| `/photo/{id}?force=like` | GET | 点赞，返回 `{code:200,msg:""}` |
| `/photo/{qiniu_url}?force=download` | GET | Bing 原图流式下载并增加下载量 |
| `/v1` 或 `/v1/` | GET / POST | 最新图片；`d=0` 当天、`d=1` 昨天；省略 d 取最新已发布记录 |
| `/v1/rand` | GET / POST | 随机已发布图片，支持有 ID 空洞的表 |
| `/v1/blur` | GET / POST | Go 内存模糊和居中裁剪；`r` 默认 10，限制 1–50 |
| `/about.html` / `/static/about.html` | GET | 关于页面 |
| `/healthz` | GET | 数据库连接健康检查 |

`/v1` 与 `/v1/rand` 默认返回图片内容，保留原接口使用方式；`w`、`h` 默认 `1366`、`768`。原有常用分辨率由 Bing 提供，其他分辨率由 Go 获取原图后在内存中居中裁剪。图片尺寸不保证是更高分辨率的原生摄影源。

支持 query、POST 表单和 POST JSON，query 优先。`type=json` 返回 JSON，`callback=app.done` 返回 JSONP（仅允许合法 JS 标识符链）。`type=redirect` 新增为 302 到 Bing，适合减少本站图片转发带宽。

```text
/v1?type=json
/v1?d=1&w=1920&h=1080
/v1/rand?type=redirect&w=1920&h=1080
/v1/blur?w=640&h=360&r=10
```

JSON 保留原 envelope 和字段：

```json
{
  "data": {
    "enddate": "20261006",
    "url": "https://www.bing.com/th?id=OHR.Example_1366x768.jpg",
    "bmiddle_pic": "https://www.bing.com/th?id=OHR.Example_800x480.jpg",
    "original_pic": "https://www.bing.com/th?id=OHR.Example_1920x1080.jpg",
    "thumbnail_pic": "https://www.bing.com/th?id=OHR.Example_400x240.jpg"
  },
  "status": {"code": 200, "message": ""}
}
```

差异：无数据返回 HTTP 404 和 `status.code=404`，参数错误返回 400，上游图片失败返回 502；数据库错误返回 500。原版部分错误会挂起、重定向循环或返回 HTTP 200，现在返回明确错误。图片只筛选配置语言且不显示未来日期。

图片转发不落盘，设置 30 秒上游超时，单图最多 20 MiB。本地裁剪/模糊限制为 4,000,000 输出像素、4 个并发；普通 Bing 原图请求最多 8,294,400 像素。动态处理会消耗 CPU；高频调用宜缓存响应。JSON 和 redirect 使用 Bing 地址，Bing 对非标准尺寸/部分历史图片可能返回 404；需要任意尺寸裁剪时使用默认图片响应。

点赞继续使用 Cookie 防止同一浏览器重复操作，这是浏览器级限制，可清除 Cookie，不是账户级去重。计数改为原子 `COALESCE(column,0)+1`；下载完成后才统计下载量。页面样式保留，使用原生懒加载与 JS，移除了 jQuery 依赖和禁止开发者工具的脚本。

## 可选邮件和微博

SMTP 不配置 `address` 时不发送通知。配置后同步失败或新增记录会发送通知，重复同步不发“失败”邮件。示例 `address:"smtp.qq.com:465"`，填写发件人、收件人、用户名和授权码。支持 465 TLS 和 STARTTLS，不关闭证书校验。

微博为可选模块；默认未配置时返回 503。填写 `client_id`、`client_secret`、至少 32 字符随机 `session_secret` 及 HTTPS `root_url`。微博应用回调 URL 设置为 `{root_url}/weibo/callback`。保留 `bing_session` 表，`insertdate` 改写 Unix 秒（原 JS 毫秒值无法放入 INT），旧授权需重新登录。

`/weibo` 发起 OAuth；回调验证 state 并将 token 保存原表，使用签名 HttpOnly Cookie 绑定当前 UID，避免原实现的全局 token 串号。`/weibo/shorten?url=...` 保留短链接口；`GET /weibo/send` 显示确认表单，实际发送使用带 CSRF 的 POST。自动发微博仍保持关闭，符合原版注释状态。微博是否可发送/生成短链还取决于应用权限。

## systemd（Debian）

停止旧 Node 服务，将二进制和配置放到 `/opt/bing`：

```sh
sudo useradd --system --no-create-home --shell /usr/sbin/nologin bing
sudo install -d /opt/bing
sudo install -m 755 bing /opt/bing/bing
sudo install -o root -g bing -m 640 config.yaml /opt/bing/config.yaml
sudo install -m 644 bing.service /etc/systemd/system/bing.service
sudo systemctl daemon-reload
sudo systemctl enable --now bing
sudo journalctl -u bing -f
```

如果已存在 bing 用户可跳过 useradd。原反代端口仍为 12345，可继续使用 Caddy/nginx。默认监听所有网卡；只经反代访问时，可配置 `listen:"127.0.0.1:12345"`。

## Docker

```sh
docker build -t bing-go .
docker run -d --name bing-go --restart unless-stopped \
  -p 12345:12345 \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  bing-go
```

容器内的 MySQL host 应指向数据库服务地址，不能用宿主机的 127.0.0.1。运行用户为 UID 10001，挂载配置需允许该用户读取。

## 验证

```sh
go test -race ./...
go vet ./...
# 可选：在可创建/删除数据库的隔离测试实例上跑原表结构集成测试
BING_TEST_DSN='root@tcp(127.0.0.1:13306)/' go test -race -count=1 -v ./...
```

集成测试创建并删除自己命名的 `bing_test_*` 临时数据库，在该库导入原 SQL，验证 MyISAM 未变、4 实例并发去重、旧链接和 OSS 地址兼容、NULL 计数、随机 ID 空洞、页面、流式下载、本地裁剪/模糊和空库响应。不会导入或删除 DSN 指定数据库内的数据。
