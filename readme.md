
# Bing Pictures Interface | 必应壁纸接口
> `Doem:`  https://bing.bfsea.xyz

> `使用方法参见原作:` https://github.com/xCss/bing/

# 部署方法

注意 : 不支持Windows环境

下载源码&安装node和npm

```bash
apt-get install nodejs npm
```

cd到源码目录&执行依赖安装
```bash
cd bing/
npm install
```
导入sql文件到你的数据库`bing.sql`

执行一下sql语句清空ID缓存(可选)
```sql
truncate table 库名;
```
编译`configs/config.js`配置文件
```javascript
module.exports = {
    bing_env: /*运行环境*/ {
        PORT: '12345',                          //监听端口
        SERT: 'abcdefg',                        //随机字符即可
        BURL: 'https://www4.bing.com/',         //BING的地址
        BCDN: 'https://v1.cdn.bfsea.xyz/bing/', //你的CDN
        ROOT: 'https://bing.bfsea.xyz/',        //你的根目录
        LANG: 'zh-cn'
    },
    qiniu_dev: /*七牛环境*/ {
        AK: 'aaaaaaaaaaaaaaaaaaa',  //你的七牛AK
        SK: 'bbbbbbbbbbbbbbbbbbb',  //你的七牛SK
        BUCKET: 'ccccc'             //你的七牛空间
    },
    mail_dev: /*邮件环境*/ {
        SERV: 'QQ',             //服务类型
        USER: 'abc@qq.com',     //发件人
        PASS: 'asdfghjkl',      //发件人密码
        RECV: 'def@qq.com'      //收件人
    },
    mysql_dev: /*数据库环境*/ {
        host: 'localhost',      //地址
        database: 'bing',       //库名
        user: 'bing',           //用户
        password: 'bing',       //密码
        port: 3306,             //端口
        connectionLimit: 10,
        supportBigNumbers: true,
        multipleStatements: true,
        insecureAuth: true
    },
......

```

安装`supervisor`
```bash
npm install -g supervisor
```
启动项目
```bash
supervisor node bin/www
```
