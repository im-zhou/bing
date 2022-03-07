var request = require('superagent');
var objectAssign = require('object-assign');
var qiniuUtils = require('./qiniuUtils');
var commonUtils = require('./commonUtils');
var config = require('../configs/config');
var BIND_URL = config.bing_env.BURL;
var LANGMKT = config.bing_env.LANG;
var CDN = config.bing_env.BCDN;
var bingURL = BIND_URL + 'HPImageArchive.aspx';
var cookie = { 'user-agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36 Edg/95.0.1020.44' };
module.exports = {
    /**
     * 获取 当日Bing图片
     */
    fetchPicture: function (options, callback) {
        console.log('fetchPicture no s!')
        var defaultOptions = {
            ids: 0,
            n: 1,
            format: 'js',
            mkt: LANGMKT
        };
        if (Object.prototype.toString.call(options) === '[object Object]') {
            // 合并对象
            defaultOptions = objectAssign(defaultOptions, options);
        } else {
            callback = options;
        }
        request
            .get(bingURL)
            .set(cookie)
            .query(defaultOptions)
            .end(function (err, res) {
                commonUtils.convert(err, res, function (data) {
                    for (var i in data['images']) {
                        var images = data['images'][i];
                        console.log('fetchPicture starting!'),
                        console.log(data),
                        console.log('fetchPicture Progressing_1!'),
                            //data = objectAssign(images, data);
                            data = objectAssign(images, data)
                            console.log('fetchPicture Progressing_2!')
                            var newData = {
                                startdate: data.startdate,
                                fullstartdate: data.fullstartdate,
                                enddate: data.enddate,
                                url: CDN + data.urlbase.replace("/th?id=", "") + '_1920x1080.jpg',
                                urlbase: data.urlbase,
                                copyright: data.copyright,
                                copyrightlink: data.copyrightlink,
                                hsh: data.hsh,
                                title: data.title.replace('"',"~"),
                                description: data.description,
                                attribute: data.attribute,
                                country: data.country,
                                city: data.city,
                                longitude: data.longitude,
                                latitude: data.latitude,
                                continent: data.continent,
                                mkt: LANGMKT,
                                qiniu_url: data.urlbase.replace("/th?id=", ""),
                                fetch_url: BIND_URL + data.urlbase.replace("/", "") + '_1920x1080.jpg',
                            }
                            // 抓取后上传
                            qiniuUtils.fetchToQiniu(newData.fetch_url)
                            console.log('=====bingUtils fetchToQiniue Done!=====')
                            callback && callback(newData);
                            console.log('=====返回的数据START=====')
                            console.log(newData)
                            console.log('=====返回的数据END=====')
                            console.log('=====fetchPicture Done End!=====')
                    }
                });
            });
    },
    /**
     * 获取 当前Bing返回的所有图片集合
     */
    fetchPictures: function (callback) {
        console.log('fetchPictures!')
        var options = {
            ids: 14,
            n: 100
        };
        module.exports.fetchPicture(options, callback);
    },
};
