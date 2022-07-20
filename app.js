const express = require('express');
const path = require('path');
const favicon = require('serve-favicon');
const logger = require('morgan');
const cookieParser = require('cookie-parser');
const bodyParser = require('body-parser');
const session = require('express-session');
// 路由
const index = require('./routes/index');
const photo = require('./routes/photo');
const v1 = require('./routes/v1');
// 定时器
const schedule = require('node-schedule');

// 所有类文件
const config = require('./configs/config');
const fetchToday = require('./utils/fetchToday')

// URL
const ROOT = config.bing_env.ROOT;

// Get Bing data at 01:00 every day
schedule.scheduleJob('*/60 * * * *', () => {
    fetchToday.getToday()
});

const app = express();
app.disable('x-powered-by');
app.use('/static', express.static(path.join(__dirname, 'static')));
app.use(favicon(__dirname + '/static/images/bing.ico'));
// view engine setup
app.set('views', path.join(__dirname, 'views'));
app.set('view engine', 'pug');
app.enable('trust proxy');
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({
    extended: true
}));
app.use(cookieParser(ROOT));
app.use(session({
    secret: config.bing_env.SERT, //secret的值建议使用随机字符串
    resave: true,
    saveUninitialized: false,
    cookie: {
        secure: true,
        maxAge: 60 * 30 * 1000 // 过期时间（毫秒）
    }
}));
// 设置日志
app.use(logger('combined', {}));

/**
 * 处理OPTIONS请求
 */
app.use(function (req, res, next) {
    if (req.method === 'OPTIONS') {
        res.sendStatus(200);
    } else next();
});

app.use('/', index);
app.use('/photo', photo);
app.use('/v1', v1);

app.get('/about.html', function (req, res, next) {
    res.render('about');
});

/**
 * Robots.txt
 */
app.get('/robots.txt', function (req, res, next) {
    res.header('content-type', 'text/plain');
    res.send('User-Agent: * \nAllow: /');
});

/**
 * 防止拉取配置文件
 */
app.get('/configs/config.js', function (req, res, next) {
    //req.destroy();
    res.redirect('/')
});

// catch 404 and forward to error handler
app.use(function (req, res, next) {
    var err = new Error('啊( ⊙ o ⊙ )，你发现了新大陆 ∑(っ °Д °;)っ');
    err.status = 404;
    next(err);
});
// error handlers
// development error handler
// will print stacktrace
if (app.get('env') === 'development') {
    app.use(function (err, req, res, next) {
        res.status(err.status || 500);
        res.render('error', {
            message: err.message,
            error: err
        });
    });
}
// production error handler
// no stacktraces leaked to user
app.use(function (err, req, res, next) {
    res.status(err.status || 500);
    res.render('error', {
        message: err.message,
        error: {}
    });
});
module.exports = app;
