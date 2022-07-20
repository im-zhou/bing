const date = require("./date");
const dbUtils = require("./dbUtils");
const bingUtils = require("./bingUtils");
const mail = require("./mailUtils");
module.exports = {
    getToday: function () {
        console.log('Time:' + date.getTime())
        // 查询是否已经抓取并插入数据库，如果已插入就不重复抓取
        dbUtils.get('bing', {
            body: {
                enddate: date.getNow()
            }
        }, function (rows) {
            //console.info('返回数据:', rows['rows'])
            if (rows['rows'].length === 0) {
                bingUtils.fetchTodayPicture({}, function (data) {
                    dbUtils.set('bing', data, function (rows) {
                        console.info('Succeeded in inserting the database: ', rows)
                        mail.send({
                            message: 'Bing Global',
                            title: 'Bing Global',
                            stack: data,
                        });
                    })
                });
            } else {
                console.error('Data already exists')
            }
        })
    }
}
