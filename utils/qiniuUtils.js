var qiniu = require('qiniu');
var config = require('../configs/config');
var CDN = config.bing_env.BCDN;
// access_key and secret_key
const ACCESS_KEY = config.qiniu_dev.AK;
const SECRET_KEY = config.qiniu_dev.SK;
const mac = new qiniu.auth.digest.Mac(ACCESS_KEY, SECRET_KEY);
const qiniuConfig = new qiniu.conf.Config();
// 上传的空间
var bucket = config.qiniu_dev.BUCKET;

module.exports = {
    /**
     * 上传到骑牛
     * @param imgUrl 远程图片地址
     * @param callback
     */
    fetchToQiniu: function (imgURL, callback) {
        console.log('qiniuutils fetch_1')
        let bucketManager = new qiniu.rs.BucketManager(mac, qiniuConfig);
        for (var i = 0, len = config.resolutions.length; i < len; i++) {
            var _temp = config.resolutions[i];
            var remoteURL = imgURL.replace('1920x1080', _temp);
            //console.log(imgURL);
            var _tempName = 'bing/' + imgURL.substr(imgURL.lastIndexOf('=') + 1, imgURL.length);
            var imgName = _tempName.replace('1920x1080', _temp);
            bucketManager.fetch(remoteURL, bucket, imgName, function (err, ret, info) {
                if (err) {
                    console.log(err);
                } else {
                    console.log(ret + info);
                }
            });
        }
        console.log('qiniuutils fetch_Done')
        callback && callback();
    },
    /**
     * 创建预览图片
     * @param {String} url      图片名称
     * @param {Number} width    缩略图宽度(默认1920)
     * @param {Number} height   缩略图高度(默认1080)
     * @param {Number} quality  图片质量(0-100,默认100)
     * @param {Number} mode     缩略图处理模式(默认1)
     *                              1:表示限定目标缩略图的宽度和高度，放大并从缩略图中央处裁剪为指定 <Width>x<Height> 大小的图片。
     *                              2:指定 <Width> 和 <Height>，表示限定目标缩略图的长和宽，将缩略图的大小限定在指定的宽高矩形内。
     *                              2:指定 <Width> 但不指定 <Height>，表示限定目标缩略图的宽度，高度等比缩略自适应。
     *                              2:指定 <Height> 但不指定 <Width>，表示限定目标缩略图的高度，宽度等比缩略自适应。
     * @return {String} fullURL
     * 
     */
    imageView: function (url, width, height, quality, mode) {
        width = width || 1920;
        height = height || 1080;
        quality = quality || 100;
        mode = mode || 1;
        url = url.indexOf('1920x1080') == -1 ? url + '_1920x1080.jpg' : url;
        url = /^(http|https)/.test(url) ? url : CDN + url;
        return `${url}?imageView2/${mode}/w/${width}/h/${height}/format/jpg/interlace/1/q/${quality}`
    },
};