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
    weibo: /*微博环境*/ {
        /* 没用过不知道 */
        CLIENT_ID: '',
        CLIENT_SECRET: '',
        ACCESS_TOKEN: '',
        MASTER_ACCESS_TOKEN: '',
        MASTER_UID: '',
        USER_UID: ''
    },
    //disabled: [process.env.disabled.split(',')],
    /**
     * 已知分辨率
     */
    resolutions: [
        '1920x1200',
        '1920x1080',
        '1366x768',
        '1280x768',
        '1080x1920',
        '1024x768',
        '800x600',
        '800x480',
        '768x1366',
        '768x1280',
        '768x1024',
        '720x1280',
        '640x480',
        '640x360',
        '480x800',
        '400x240',
        '320x240',
        '320x180',
        '240x400',
        '240x320',
        '240x240',
        '200x200',
        '150x150'
    ],
}
