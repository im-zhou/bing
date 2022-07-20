module.exports = {
    bing_env: /*运行环境*/ {
        PORT: '8082',                          //监听端口
        SERT: 'iawdbk2ldanwkdn',                        //随机字符即可
        BURL: 'https://www.bing.com/',         //BING的地址
        BCDN: 'https://oss-chuncheon.bfsea.xyz/bing/', //你的CDN
        ROOT: 'https://qemon.tk/',        //你的根目录
        LANG: 'en-US'
    },
    mail_dev: /*邮件环境*/ {
        SERV: '163',             //服务类型
        USER: 'admin@example.com',     //发件人
        PASS: 'password',      //发件人密码
        RECV: 'recv@example.com',      //收件人
        HOST: 'smtp.example.com',
        PORT: 587,
    },
    Oracle_DB: {
        libDir: '/home/instantclient_21_6/',
        user: 'ADMIN',
        pass: 'PASSWORD',
        conn: `YOUR_CONNECT_STRING`
    },
    Oracle_Oss: {
        confPath: '/home/OracleAccess/access.txt',
        confProfile: 'DEFAULT',
        namespace: 'YOUR_NAMESPACE',
        bucket: 'YOUR_BUCKET',
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
