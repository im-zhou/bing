const request = require('superagent');
const objectAssign = require('object-assign');
const OssUtils = require('./OssUtils');
const commonUtils = require('./commonUtils');
const config = require('../configs/config');
const BIND_URL = config.bing_env.BURL;
const LANGMKT = config.bing_env.LANG;
const CDN = config.bing_env.BCDN;
const bingURL = BIND_URL + 'HPImageArchive.aspx';
const cookie = {'user-agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36 Edg/95.0.1020.44'};
module.exports = {
    /**
     * 获取 当日Bing图片
     */
    fetchTodayPicture: function (options, callback) {
        let defaultOptions = {
            ids: 0,
            n: 1,
            format: 'js',
            ensearch: 1, // set lang
            mkt: LANGMKT
        };
        if (Object.prototype.toString.call(options) === '[object Object]') {
            // 合并对象
            defaultOptions = objectAssign(defaultOptions, options);
        } else {
            callback = options;
        }
        //console.warn('参数: ', defaultOptions)
        request
            .get(bingURL)
            .set(cookie)
            .query(defaultOptions)
            .end(function (err, res) {
                commonUtils.convert(err, res, function (body) {
                    const arr = body['images']
                    for (const i in arr) {
                        const data = arr[i];
                        //console.info('本次数据: ', data['enddate'].toString())
                        const newData = {
                            startdate: data.startdate,
                            fullstartdate: data.fullstartdate,
                            enddate: data.enddate,
                            url: CDN + data.urlbase.replace("/th?id=", "") + '_1920x1080.jpg',
                            urlbase: data.urlbase,
                            copyright: data.copyright.replaceAll('\'', '').replaceAll('\"', ''),
                            copyrightlink: data.copyrightlink,
                            hsh: data.hsh,
                            title: data.title.replaceAll('\'', '').replaceAll('\"', ''),
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
                        };
                        // 抓取后上传
                        OssUtils.fetchToOss(newData.fetch_url)
                        callback && callback(newData);
                    }
                });
            });
    },
    /**
     * 获取 7天内Bing返回的所有图片集合
     */
    fetch7DaysPictures: function (callback) {
        const options = {
            n: 8
        };
        module.exports.fetchTodayPicture(options, callback);
    },
};
