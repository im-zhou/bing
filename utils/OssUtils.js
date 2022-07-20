const config = require('../configs/config');
const CDN = config.bing_env.BCDN;
const oracle_oss = require('./oracle_oss')

module.exports = {
    /**
     * 上传到OSS
     * @param imgURL 图片链接
     * @param callback
     */
    fetchToOss: function (imgURL, callback) {
        for (let i = 0, len = config.resolutions.length; i < len; i++) {
            const _temp = config.resolutions[i];
            const remoteURL = imgURL.replace('1920x1080', _temp);
            const _tempName = 'bing/' + imgURL.substring(imgURL.lastIndexOf('=') + 1, imgURL.length);
            const imgName = _tempName.replace('1920x1080', _temp);
            //console.info('目标文件: ', remoteURL + '文件名: ', imgName)
            oracle_oss.fetchToOracleOss(remoteURL, imgName, function (response) {
                console.info('上传结束', response['opcRequestId'])
            })
        }
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
