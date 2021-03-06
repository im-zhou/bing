var express = require('express');
var path = require('path');
var favicon = require('serve-favicon');
var logger = require('morgan');
var cookieParser = require('cookie-parser');
var bodyParser = require('body-parser');
var session = require('express-session');
var request = require('superagent');
// 路由
var index = require('./routes/index');
var photo = require('./routes/photo');
var weibo = require('./routes/weibo');
var v1 = require('./routes/v1');
// 定时器
var schedule = require('node-schedule');

// 所有类文件
var dbUtils = require('./utils/dbUtils');
var bingUtils = require('./utils/bingUtils');
var mailUtils = require('./utils/mailUtils');
var qiniuUtils = require('./utils/qiniuUtils');
var weiboUtils = require('./utils/weiboUtils');
var config = require('./configs/config');

// URL
var ROOT = config.bing_env.ROOT;

var app = express();
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
app.use(logger('combined', {
    skip: function(req, res) { return res.statusCode < 400 }
}));
/*
 * 手动更新 
 * /fetch?url=https://www4.bing.com/th?id=OHR.MeotoIwa_ZH-CN3126370410_1920x1080.jpg
*/

// app.get('/fetch',function(req,res,next){
//     console.log('Manual fetch!');
//         if(req.query.url){
//             console.log('appJS.fetch 手动抓取!'),
//             qiniuUtils.fetchToQiniu(req.query.url)
//             res.redirect('/');
//         }else{
//             console.log('appJS.fetch.手动抓取失败!'),
//             //console.log(url)
//             res.redirect('/');
//         }
// })

// 每天 01:00 检测bing数据
schedule.scheduleJob('0 1 * * *', function() {
    var date = new Date((new Date()).getTime())
    var year = date.getFullYear();
    var month = date.getMonth() + 1;
    var day = date.getDate();
    var hour = (date.getHours() < 10 ? '0' + date.getHours() : date.getHours()) + ':';
    var minute = (date.getMinutes() < 10 ? '0' + date.getMinutes() : date.getMinutes()) + ':';
    var second = (date.getSeconds() < 10 ? '0' + date.getSeconds() : date.getSeconds());
    var now = year + '' + (month < 10 ? '0' + month : month) + '' + (day < 10 ? '0' + day : day);
    var time = now + '#' + hour + minute + second;
    console.log('更新:' + time)
    // 查询是否已经抓取并插入数据库，如果已插入就不重复抓取
    dbUtils.get('bing', {
        body: {
            enddate:now
        }
     }, function(rows) {
         if (rows.length === 0) {
            bingUtils.fetchPicture({}, function(data) {
                dbUtils.set('bing', data, function(rows) {
                    data.id = rows.insertId || 0;
                    mailUtils.send({
                        message: '从Bing抓取成功',
                        title: '从Bing抓取成功',
                        message: '系统时间：' + time,
                        stack: JSON.stringify(data, '', 4),
                    });
                })
            });
        } else {
			mailUtils.send({
                message: '从Bing抓取失败,或数据已经存在!',
                title: '从Bing抓取失败,或数据已经存在!',
                message: '系统时间：' + time,
            });
		}
    });
});
// 每天 08:30,12:30,15:30,18:30,21:30 定时发送微博
//schedule.scheduleJob('*/30 8-21 * * *', function() {
    //weiboUtils.update(function(data) {
        //if (data && data.id) {
            // mailUtils.send({
            //     message: '发送微博成功',
            //     title: '发送微博成功',
            //     stack: JSON.stringify(data, '', 4)
            // });
        //} else {
            //mailUtils.send({
                //message: '发送微博失败',
                //title: '发送微博失败',
                //stack: JSON.stringify(data, '', 4)
            //});
        //}
    //}, true);
//});

/**
 * 处理OPTIONS请求
 */
app.use(function(req, res, next) {
    if (req.method === 'OPTIONS') {
        res.sendStatus(200);
    } else next();
});

app.use('/', index);
app.use('/photo', photo);
app.use('/weibo', weibo);
app.use('/v1', v1);

app.get('/about.html', function(req, res, next) {
    res.render('about');
});

/**
 * Robots.txt
 */
app.get('/robots.txt', function(req, res, next) {
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
app.use(function(req, res, next) {
    var err = new Error('啊( ⊙ o ⊙ )，你发现了新大陆 ∑(っ °Д °;)っ');
    err.status = 404;
    next(err);

    //res.redirect('/');
});
// error handlers
// development error handler
// will print stacktrace
if (app.get('env') === 'development') {
    app.use(function(err, req, res, next) {
        res.status(err.status || 500);
        res.render('error', {
            message: err.message,
            error: err
        });
    });
}
// production error handler
// no stacktraces leaked to user
app.use(function(err, req, res, next) {
    res.status(err.status || 500);
    res.render('error', {
        message: err.message,
        error: {}
    });
});
module.exports = app;
