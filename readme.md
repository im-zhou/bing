
# Bing Pictures Interface | 必应壁纸接口
> `Doem:`  http://bing.bfsea.xyz

> `使用方法参见原作:` https://github.com/xCss/bing/

# 部署方法

注意 : 不支持Windows环境

下载源码&安装node和npm
```
apt-get install nodejs npm
```
cd到源码目录&执行依赖安装
```
npm install
```
导入sql文件到你的数据库`bing.sql`

执行一下sql语句清空ID缓存
```
truncate table 库名;
```
编译`env.txt`变量文件
```
export BING_PORT=8081
export BING_CDN=https://v1.cdn.bfsea.xyz/bing/
export BING_ROOT=https://bing.bfsea.xyz/
export WEIBO_CLIENT_ID=''
export WEIBO_CLIENT_SECRET=''
export WEIBO_ACCESS_TOKEN=''
export WEIBO_MASTER_ACCESS_TOKEN=''
export WEIBO_MASTER_UID=''
export WEIBO_USER_UID=''
export MYSQL_HOST=127.0.0.1
export MYSQL_DATABASE=bing
export MYSQL_USER=user
export MYSQL_PASSWORD=pass
export MYSQL_PORT=3306
export MAIL_USER=发件人@vip.qq.com
export MAIL_PASSWORD=yourpassword
export MAIL_RECIPIENT=接收人@qq.com
export MAIL_SERVICE=QQ
export QINIU_ACCESS_KEY=yourkey
export QINIU_SECRET_KEY=yourkey
export QINIU_BUCKET=yourbucket

```
应用变量
```
source env.txt
```
应用完记得删除这个文件(确保安全)

安装`supervisor`
```
npm install -g supervisor
```
启动项目
```
supervisor node bin/www
```
