
# Bing Pictures Interface | 必应壁纸接口
> `Bing 壁纸 Api` : http://bing.bfsea.xyz
# 使用方法

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

# 以下为原文

## 目前开放的壁纸接口：
 - `/v1{d,w,h,p,size,callback}` 返回今日的壁纸完整数据(`可选参数{d,w,h,p,size,callback}`)： 

    > 若指定参数`{w,h}` ，则直接返回图片

|参数名|类型|是否必要|备注|
|:----:|:---------:|:--------:|---|
|d|`Int`|否|自今日起第`d`天前的数据|
|w|`Int`|否|图片宽度|
|h|`Int`|否|图片高度|
|p|`Int`|否|`Page 页码`:第x页|
|size|`Int`|否|`Size 条数`:每页条数|
|callback|`String`|否|JSONP的回调函数名|

 - `/v1/rand{w,h,type,callback}` 返回随机的壁纸(`可选参数{w,h,type,callback}`)：

|参数名|类型|是否必要|备注|
|:----:|:---------:|:--------:|---|
|w|`Int`|否|图片宽度|
|h|`Int`|否|图片高度|
|type|`String`|否|返回值类型(`json`)|
|callback|`String`|否|JSONP的回调函数名|

- `/v1/blur{d,w,h,r}` 返回高斯模糊壁纸(`可选参数{d,w,h,r}`)：

|参数名|类型|是否必要|备注|
|:----:|:---------:|:--------:|---|
|d|`Int`|否|自今日起第`d`天前的数据|
|w|`Int`|否|图片宽度|
|h|`Int`|否|图片高度|
|r|`Int`|否|模糊半径(`1~50`)|

### **:warning:** `高斯模糊`接口目前只支持指定分辨率(`w,h`)的图片，具体分辨率如下：
```js
/**
 * 已知分辨率
 */
resolutions: [
    '1920x1200',
    '1920x1080',
    '1366x768',
    '1280x768',
    '1024x768',
    '800x600',
    '800x480',
    '768x1280',
    '720x1280',
    '640x480',
    '480x800',
    '400x240',
    '320x240',
    '240x320'
]
```
