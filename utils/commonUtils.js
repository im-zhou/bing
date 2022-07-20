var mailUtils = require('./mailUtils');
module.exports = {
    /**
     * 公共转换函数
     * @err         错误信息
     * @res         响应信息
     * @callback    回调函数
     */
    convert: function (err, res, callback) {
        //console.log('=====响应信息处理=====')
        try {
            if (!err && res.status === 200) {
                var error_code = null;
                var body = null;
                if (res && res.text) {
                    body = res.text;
                }
                if (typeof body === 'string') {
                    try {
                        body = JSON.parse(body);
                    } catch (error) {
                        throw new Error(error);
                    }
                }
                if (body.error_code || body.error) {
                    throw new Error(res);
                } else {
                    callback && callback(body);
                }
            } else {
                throw new Error(res);
            }
        } catch (error) {
            // send mail
            //console.log(error.message);
            console.log(error);
            mailUtils.send(error);
        }
    }
}
