module.exports = {
    weibo: {
        CLIENT_ID: process.env.WEIBO_CLIENT_ID,
        CLIENT_SECRET: process.env.WEIBO_CLIENT_SECRET,
        ACCESS_TOKEN: process.env.WEIBO_ACCESS_TOKEN,
        MASTER_ACCESS_TOKEN: process.env.WEIBO_MASTER_ACCESS_TOKEN,
        MASTER_UID: process.env.WEIBO_MASTER_UID,
        USER_UID: process.env.WEIBO_USER_UID
    },
    mysql_dev: {
        host: process.env.MYSQL_HOST,
        database: process.env.MYSQL_DATABASE,
        user: process.env.MYSQL_USER,
        password: process.env.MYSQL_PASSWORD,
        port: process.env.MYSQL_PORT,
        connectionLimit: 10,
        supportBigNumbers: true,
        multipleStatements: true,
        insecureAuth: true
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
